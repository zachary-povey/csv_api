package parquet_writer

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/parquet-go/parquet-go"
	"github.com/zachary-povey/csv_api/internal/config"
	"github.com/zachary-povey/csv_api/internal/error_tracking"
)

// rowsPerRowGroup bounds how many rows are buffered in memory before a row
// group is written out. A variable only so tests can use small row groups.
var rowsPerRowGroup = 100_000

// writeBufferSize is the size of the buffer between the Parquet writer and
// the output; it is flushed after each row group.
const writeBufferSize = 64 * 1024

// orderedGroup is a group node whose fields keep the config's order
// (parquet.Group sorts its fields by name).
type orderedGroup struct {
	parquet.Group
	fields []parquet.Field
}

func (g orderedGroup) Fields() []parquet.Field { return g.fields }

// columnNode maps a field's logical type to a Parquet leaf node, equivalent
// to the Avro output (see map_type_json in avro_writer).
func columnNode(field config.FieldConfig) parquet.Node {
	switch ltc := field.LogicalTypeConfig.(type) {
	case config.IntegerTypeConfig:
		return parquet.Int(64)
	case config.StringTypeConfig:
		return parquet.String()
	case config.DecimalTypeConfig:
		if ltc.Args.AsFloat {
			return parquet.Leaf(parquet.DoubleType)
		}
		// like Avro's bytes decimal: big-endian two's complement unscaled value
		return parquet.Decimal(ltc.Args.Scale, ltc.Args.Precision, parquet.ByteArrayType)
	case config.EnumTypeConfig:
		// Parquet's ENUM annotation has no list of symbols, so the permitted
		// values are checked by the parser but not recorded in the schema
		return parquet.Enum()
	case config.DateTypeConfig:
		return parquet.Date()
	case config.TimeTypeConfig:
		// a local time of day, like Avro's time-micros
		return parquet.TimeAdjusted(parquet.Microsecond, false)
	case config.TimestampTypeConfig:
		// an instant (converted to UTC), like Avro's timestamp-micros
		return parquet.TimestampAdjusted(parquet.Microsecond, true)
	case config.BooleanTypeConfig:
		return parquet.Leaf(parquet.BooleanType)
	default:
		panic(fmt.Sprintf("Unknown logical type config: %T", field.LogicalTypeConfig))
	}
}

// generateSchema builds the Parquet schema for config: one column per field,
// in config order, OPTIONAL if the field is nullable and REQUIRED otherwise.
func generateSchema(conf *config.Config) *parquet.Schema {
	fields := make([]parquet.Field, len(conf.Fields))
	for i, field := range conf.Fields {
		node := columnNode(field)
		if field.Nullable {
			node = parquet.Optional(node)
		} else {
			node = parquet.Required(node)
		}
		fields[i] = parquet.Group{field.Name: node}.Fields()[0]
	}
	return parquet.NewSchema(conf.Name, orderedGroup{fields: fields})
}

// columnValue converts a parsed value to its Parquet value. nil is null.
func columnValue(field config.FieldConfig, value any) (parquet.Value, error) {
	if value == nil {
		return parquet.NullValue(), nil
	}
	switch ltc := field.LogicalTypeConfig.(type) {
	case config.IntegerTypeConfig:
		if v, ok := value.(int); ok {
			return parquet.Int64Value(int64(v)), nil
		}
	case config.StringTypeConfig, config.EnumTypeConfig:
		if v, ok := value.(string); ok {
			return parquet.ByteArrayValue([]byte(v)), nil
		}
	case config.DecimalTypeConfig:
		if ltc.Args.AsFloat {
			if v, ok := value.(float64); ok {
				return parquet.DoubleValue(v), nil
			}
		} else if v, ok := value.(*big.Rat); ok {
			return parquet.ByteArrayValue(decimalBytes(v, ltc.Args.Scale)), nil
		}
	case config.DateTypeConfig:
		if v, ok := value.(int32); ok {
			return parquet.Int32Value(v), nil
		}
	case config.TimeTypeConfig, config.TimestampTypeConfig:
		if v, ok := value.(int64); ok {
			return parquet.Int64Value(v), nil
		}
	case config.BooleanTypeConfig:
		if v, ok := value.(bool); ok {
			return parquet.BooleanValue(v), nil
		}
	}
	return parquet.Value{}, fmt.Errorf("unexpected value %v (%T) for field '%s'", value, value, field.Name)
}

// decimalBytes returns the unscaled value of r at scale as big-endian two's
// complement bytes. The converter only produces values that fit the field's
// precision and scale (config.DecimalArg), so this is exact, as is goavro's
// encoding of the same *big.Rat in the Avro output.
func decimalBytes(r *big.Rat, scale int) []byte {
	unscaled := new(big.Int).Mul(r.Num(), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil))
	unscaled.Div(unscaled, r.Denom())
	if unscaled.Sign() >= 0 {
		return withSignByte(unscaled.Bytes())
	}
	// -x is the bitwise inverse of x-1
	b := withSignByte(new(big.Int).Sub(new(big.Int).Neg(unscaled), big.NewInt(1)).Bytes())
	for i := range b {
		b[i] = ^b[i]
	}
	return b
}

// withSignByte prepends a zero byte if b is empty or its top bit is set, so
// b reads as a non-negative two's complement number.
func withSignByte(b []byte) []byte {
	if len(b) == 0 || b[0]&0x80 != 0 {
		return append([]byte{0}, b...)
	}
	return b
}

// Write writes the rows from channel to output as a Parquet file. Each row
// group is written to output as soon as it is complete, so with --stream the
// output is written as rows are validated (in bounded memory), though a
// Parquet file is only readable once its footer (holding the schema and row
// group index) is written at the end. If the run has failed by the time the
// channel is closed, no footer is written, so incomplete output (with
// --stream) can't be read as Parquet. output need not be seekable.
func Write(output io.Writer, conf *config.Config, channel chan map[string]any, wg *sync.WaitGroup, errTracker *error_tracking.ErrorTracker) {
	defer wg.Done()

	// parquet-go's own buffer is turned off so that flushing ours after each
	// row group gets the whole row group to the output
	buffered := bufio.NewWriterSize(output, writeBufferSize)
	writer := parquet.NewWriter(buffered, generateSchema(conf), parquet.WriteBufferSize(0))

	rows := 0
	for row := range channel {
		values := make(parquet.Row, len(conf.Fields))
		for i, field := range conf.Fields {
			value, err := columnValue(field, row[field.Name])
			if err != nil {
				errTracker.AddExecutionError(fmt.Errorf("error writing parquet record: %w", err))
				return
			}
			definitionLevel := 0
			if field.Nullable && row[field.Name] != nil {
				definitionLevel = 1
			}
			values[i] = value.Level(0, definitionLevel, i)
		}
		if _, err := writer.WriteRows([]parquet.Row{values}); err != nil {
			errTracker.AddExecutionError(fmt.Errorf("error writing parquet record: %s", err))
			return
		}
		rows++
		if rows%rowsPerRowGroup == 0 {
			err := writer.Flush()
			if err == nil {
				err = buffered.Flush()
			}
			if err != nil {
				errTracker.AddExecutionError(fmt.Errorf("error writing parquet row group: %s", err))
				return
			}
		}
	}
	// every pipeline error is recorded before the parser closes the channel
	if errTracker.Failed() {
		return
	}
	err := writer.Close()
	if err == nil {
		err = buffered.Flush()
	}
	if err != nil {
		errTracker.AddExecutionError(fmt.Errorf("error writing parquet file: %s", err))
	}
}
