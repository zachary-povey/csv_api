# Logical Type Conversion

Summed up in a sentence, `csv_api` takes a csv of an arbitrary format and a config file which defines how logical types are physically represented in strings within the csv and outputs the values from the csv converted into their logical types (or an error report on failure).

To set up your config file to achieve this, you'll need to know which logical types are available and how they can be constructed. Each logical type has one or more parameter sets defined that can be used to construct an instance of said type. Each entry in `fields[*].representations` must define how to extract all required parameters for a given parameter set, either as hard coded values or regex groups. Note that representations in `fields[*].representations` can be omitted which will result in a default list of representations being used.

Below we define each logical type with it's valid parameter sets and the default reprenstations used to generate these.

## Integer
A.k.a a whole number.

### Type Args
None.

### Parameter Sets

```
(value: string) -> int
```

### Default Representations
- `(?<value>[0-9]+)`

## Decimal
A number with a fractional part - note that the output data can be a float or a true decimal depending on the type args specified.

### Type Args
There are two mutually exlucsive sets of type arguments for decimals:
- `precision: integer`
- `scale: integer = 0`

or
- `as_float: true`

### Parameter Sets
```
(value: string) -> decimal
(integer_part: string | integer, decimal_part: string | integer) -> decimal
```

## String
A string of unicode characters.

### Type Args
None.

### Parameter Sets

```
(value: string) -> string
```

### Default Representations
- `(?<value>.+)`

## Enum
A string that must match exactly one of  a set of predefined values (see type args)

### Type Args
- `permitted_values: list[string]`

### Parameter Sets
```
(value: string) -> enum
```

### Default Representations
- `(?<value>.+)`

## Timestamp
An exact point in time.

### Type Args
None.

### Parameter Sets
```
(value: [string | int], offset: [string | int] = 0, precision: [string | int]) -> timestamp
( 
    year: [string | int],
    month: [string | int],
    day: [string | int],
    hour: [string | int] = 0,
    minute: [string | int] = 0,
    second: [string | int] = 0,
    millisecond: [string | int] = 0,
    microsecond: [string | int] = 0, 
    timezone: [string | null] = null
) -> timestamp
```

### Default Representations
Default representation is iso format, i.e.:
- `^(?<year>\d{4})-(?<month>0[1-9]|1[0-2])-(?<day>0[1-9]|[12]\d|3[01])T(?<hour>[01]\d|2[0-3]):(?<minute>[0-5]\d):(?<second>[0-5]\d)(?:\.(?<fraction>\d+))?(?<offset>Z|[+-](?:[01]\d|2[0-3]):?[0-5]\d)?$`

## Time
A time of day without reference to a specific timezone.

### Type Args
None.

### Parameter Sets
```
( 
    hour: [string | int] = 0,
    minute: [string | int] = 0,
    second: [string | int] = 0,
    millisecond: [string | int] = 0,
    microsecond: [string | int] = 0, 
) -> time
```


### Default Representations
Default representation is iso format, i.e.:
- `^(?<hour>[01]\d|2[0-3]):(?<minute>[0-5]\d):(?<second>[0-5]\d)(?:\.(?<fraction>\d+))?$`

## Date
A date without reference to a specific timezone.

### Type Args
None.

### Parameter Sets
```
( 
    year: [string | int],
    month: [string | int],
    day: [string | int],
) -> date
```


### Default Representations
Default representation is iso format, i.e.:
- `^(?<year>\d{4})-(?<month>0[1-9]|1[0-2])-(?<day>0[1-9]|[12]\d|3[01])$`