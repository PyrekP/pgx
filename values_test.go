package pgx_test

import (
	"github.com/jackc/pgx"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDateTranscode(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	dates := []time.Time{
		time.Date(1990, 1, 1, 0, 0, 0, 0, time.Local),
		time.Date(1999, 12, 31, 0, 0, 0, 0, time.Local),
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local),
		time.Date(2001, 1, 2, 0, 0, 0, 0, time.Local),
		time.Date(2004, 2, 29, 0, 0, 0, 0, time.Local),
		time.Date(2013, 7, 4, 0, 0, 0, 0, time.Local),
		time.Date(2013, 12, 25, 0, 0, 0, 0, time.Local),
	}

	for _, actualDate := range dates {
		var d time.Time

		err := conn.QueryRow("select $1::date", actualDate).Scan(&d)
		if err != nil {
			t.Fatalf("Unexpected failure on QueryRow Scan: %v", err)
		}
		if !actualDate.Equal(d) {
			t.Errorf("Did not transcode date successfully: %v is not %v", d, actualDate)
		}
	}
}

func TestTimestampTzTranscode(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	inputTime := time.Date(2013, 1, 2, 3, 4, 5, 6000, time.Local)

	var outputTime time.Time

	err := conn.QueryRow("select $1::timestamptz", inputTime).Scan(&outputTime)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if !inputTime.Equal(outputTime) {
		t.Errorf("Did not transcode time successfully: %v is not %v", outputTime, inputTime)
	}

	err = conn.QueryRow("select $1::timestamptz", inputTime).Scan(&outputTime)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if !inputTime.Equal(outputTime) {
		t.Errorf("Did not transcode time successfully: %v is not %v", outputTime, inputTime)
	}
}

func TestJsonTranscode(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	if _, ok := conn.PgTypes[pgx.JsonOid]; !ok {
		return // No JSON type -- must be running against old PostgreSQL
	}

	m := map[string]string{
		"key": "value",
	}
	var outputJson map[string]string

	err := conn.QueryRow("select $1::json", m).Scan(&outputJson)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if m["key"] != outputJson["key"] {
		t.Errorf("Did not transcode json successfully: %v is not %v", outputJson["key"], m["key"])
	}
}

func TestJsonbTranscode(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	if _, ok := conn.PgTypes[pgx.JsonbOid]; !ok {
		return // No JSONB type -- must be running against old PostgreSQL
	}

	m := map[string]string{
		"key": "value",
	}
	var outputJson map[string]string

	err := conn.QueryRow("select $1::jsonb", m).Scan(&outputJson)
	if err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}
	if m["key"] != outputJson["key"] {
		t.Errorf("Did not transcode jsonb successfully: %v is not %v", outputJson["key"], m["key"])
	}
}

func mustParseCIDR(t *testing.T, s string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatal(err)
	}

	return *ipnet
}

func TestInetCidrTranscode(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	tests := []struct {
		sql   string
		value net.IPNet
	}{
		{"select $1::inet", mustParseCIDR(t, "0.0.0.0/32")},
		{"select $1::inet", mustParseCIDR(t, "127.0.0.1/32")},
		{"select $1::inet", mustParseCIDR(t, "12.34.56.0/32")},
		{"select $1::inet", mustParseCIDR(t, "192.168.1.0/24")},
		{"select $1::inet", mustParseCIDR(t, "255.0.0.0/8")},
		{"select $1::inet", mustParseCIDR(t, "255.255.255.255/32")},
		{"select $1::inet", mustParseCIDR(t, "::/128")},
		{"select $1::inet", mustParseCIDR(t, "::/0")},
		{"select $1::inet", mustParseCIDR(t, "::1/128")},
		{"select $1::inet", mustParseCIDR(t, "2607:f8b0:4009:80b::200e/128")},
		{"select $1::cidr", mustParseCIDR(t, "0.0.0.0/32")},
		{"select $1::cidr", mustParseCIDR(t, "127.0.0.1/32")},
		{"select $1::cidr", mustParseCIDR(t, "12.34.56.0/32")},
		{"select $1::cidr", mustParseCIDR(t, "192.168.1.0/24")},
		{"select $1::cidr", mustParseCIDR(t, "255.0.0.0/8")},
		{"select $1::cidr", mustParseCIDR(t, "255.255.255.255/32")},
		{"select $1::cidr", mustParseCIDR(t, "::/128")},
		{"select $1::cidr", mustParseCIDR(t, "::/0")},
		{"select $1::cidr", mustParseCIDR(t, "::1/128")},
		{"select $1::cidr", mustParseCIDR(t, "2607:f8b0:4009:80b::200e/128")},
	}

	for i, tt := range tests {
		var actual net.IPNet

		err := conn.QueryRow(tt.sql, tt.value).Scan(&actual)
		if err != nil {
			t.Errorf("%d. Unexpected failure: %v (sql -> %v, value -> %v)", i, err, tt.sql, tt.value)
			continue
		}

		if actual.String() != tt.value.String() {
			t.Errorf("%d. Expected %v, got %v (sql -> %v)", i, tt.value, actual, tt.sql)
		}

		ensureConnValid(t, conn)
	}
}

func TestNullX(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	type allTypes struct {
		s   pgx.NullString
		i16 pgx.NullInt16
		i32 pgx.NullInt32
		i64 pgx.NullInt64
		f32 pgx.NullFloat32
		f64 pgx.NullFloat64
		b   pgx.NullBool
		t   pgx.NullTime
	}

	var actual, zero allTypes

	tests := []struct {
		sql       string
		queryArgs []interface{}
		scanArgs  []interface{}
		expected  allTypes
	}{
		{"select $1::text", []interface{}{pgx.NullString{String: "foo", Valid: true}}, []interface{}{&actual.s}, allTypes{s: pgx.NullString{String: "foo", Valid: true}}},
		{"select $1::text", []interface{}{pgx.NullString{String: "foo", Valid: false}}, []interface{}{&actual.s}, allTypes{s: pgx.NullString{String: "", Valid: false}}},
		{"select $1::int2", []interface{}{pgx.NullInt16{Int16: 1, Valid: true}}, []interface{}{&actual.i16}, allTypes{i16: pgx.NullInt16{Int16: 1, Valid: true}}},
		{"select $1::int2", []interface{}{pgx.NullInt16{Int16: 1, Valid: false}}, []interface{}{&actual.i16}, allTypes{i16: pgx.NullInt16{Int16: 0, Valid: false}}},
		{"select $1::int4", []interface{}{pgx.NullInt32{Int32: 1, Valid: true}}, []interface{}{&actual.i32}, allTypes{i32: pgx.NullInt32{Int32: 1, Valid: true}}},
		{"select $1::int4", []interface{}{pgx.NullInt32{Int32: 1, Valid: false}}, []interface{}{&actual.i32}, allTypes{i32: pgx.NullInt32{Int32: 0, Valid: false}}},
		{"select $1::int8", []interface{}{pgx.NullInt64{Int64: 1, Valid: true}}, []interface{}{&actual.i64}, allTypes{i64: pgx.NullInt64{Int64: 1, Valid: true}}},
		{"select $1::int8", []interface{}{pgx.NullInt64{Int64: 1, Valid: false}}, []interface{}{&actual.i64}, allTypes{i64: pgx.NullInt64{Int64: 0, Valid: false}}},
		{"select $1::float4", []interface{}{pgx.NullFloat32{Float32: 1.23, Valid: true}}, []interface{}{&actual.f32}, allTypes{f32: pgx.NullFloat32{Float32: 1.23, Valid: true}}},
		{"select $1::float4", []interface{}{pgx.NullFloat32{Float32: 1.23, Valid: false}}, []interface{}{&actual.f32}, allTypes{f32: pgx.NullFloat32{Float32: 0, Valid: false}}},
		{"select $1::float8", []interface{}{pgx.NullFloat64{Float64: 1.23, Valid: true}}, []interface{}{&actual.f64}, allTypes{f64: pgx.NullFloat64{Float64: 1.23, Valid: true}}},
		{"select $1::float8", []interface{}{pgx.NullFloat64{Float64: 1.23, Valid: false}}, []interface{}{&actual.f64}, allTypes{f64: pgx.NullFloat64{Float64: 0, Valid: false}}},
		{"select $1::bool", []interface{}{pgx.NullBool{Bool: true, Valid: true}}, []interface{}{&actual.b}, allTypes{b: pgx.NullBool{Bool: true, Valid: true}}},
		{"select $1::bool", []interface{}{pgx.NullBool{Bool: true, Valid: false}}, []interface{}{&actual.b}, allTypes{b: pgx.NullBool{Bool: false, Valid: false}}},
		{"select $1::timestamptz", []interface{}{pgx.NullTime{Time: time.Unix(123, 5000), Valid: true}}, []interface{}{&actual.t}, allTypes{t: pgx.NullTime{Time: time.Unix(123, 5000), Valid: true}}},
		{"select $1::timestamptz", []interface{}{pgx.NullTime{Time: time.Unix(123, 5000), Valid: false}}, []interface{}{&actual.t}, allTypes{t: pgx.NullTime{Time: time.Time{}, Valid: false}}},
		{"select $1::timestamp", []interface{}{pgx.NullTime{Time: time.Unix(123, 5000), Valid: true}}, []interface{}{&actual.t}, allTypes{t: pgx.NullTime{Time: time.Unix(123, 5000), Valid: true}}},
		{"select $1::timestamp", []interface{}{pgx.NullTime{Time: time.Unix(123, 5000), Valid: false}}, []interface{}{&actual.t}, allTypes{t: pgx.NullTime{Time: time.Time{}, Valid: false}}},
		{"select 42::int4, $1::float8", []interface{}{pgx.NullFloat64{Float64: 1.23, Valid: true}}, []interface{}{&actual.i32, &actual.f64}, allTypes{i32: pgx.NullInt32{Int32: 42, Valid: true}, f64: pgx.NullFloat64{Float64: 1.23, Valid: true}}},
	}

	for i, tt := range tests {
		actual = zero

		err := conn.QueryRow(tt.sql, tt.queryArgs...).Scan(tt.scanArgs...)
		if err != nil {
			t.Errorf("%d. Unexpected failure: %v (sql -> %v, queryArgs -> %v)", i, err, tt.sql, tt.queryArgs)
		}

		if actual != tt.expected {
			t.Errorf("%d. Expected %v, got %v (sql -> %v, queryArgs -> %v)", i, tt.expected, actual, tt.sql, tt.queryArgs)
		}

		ensureConnValid(t, conn)
	}
}

func TestArrayDecoding(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	tests := []struct {
		sql    string
		query  interface{}
		scan   interface{}
		assert func(*testing.T, interface{}, interface{})
	}{
		{
			"select $1::bool[]", []bool{true, false, true}, &[]bool{},
			func(t *testing.T, query, scan interface{}) {
				if reflect.DeepEqual(query, *(scan.(*[]bool))) == false {
					t.Errorf("failed to encode bool[]")
				}
			},
		},
		{
			"select $1::int[]", []int32{2, 4, 484}, &[]int32{},
			func(t *testing.T, query, scan interface{}) {
				if reflect.DeepEqual(query, *(scan.(*[]int32))) == false {
					t.Errorf("failed to encode int[]")
				}
			},
		},
		{
			"select $1::text[]", []string{"it's", "over", "9000!"}, &[]string{},
			func(t *testing.T, query, scan interface{}) {
				if reflect.DeepEqual(query, *(scan.(*[]string))) == false {
					t.Errorf("failed to encode text[]")
				}
			},
		},
		{
			"select $1::timestamp[]", []time.Time{time.Unix(323232, 0), time.Unix(3239949334, 00)}, &[]time.Time{},
			func(t *testing.T, query, scan interface{}) {
				if reflect.DeepEqual(query, *(scan.(*[]time.Time))) == false {
					t.Errorf("failed to encode time.Time[] to timestamp[]")
				}
			},
		},
		{
			"select $1::timestamptz[]", []time.Time{time.Unix(323232, 0), time.Unix(3239949334, 00)}, &[]time.Time{},
			func(t *testing.T, query, scan interface{}) {
				if reflect.DeepEqual(query, *(scan.(*[]time.Time))) == false {
					t.Errorf("failed to encode time.Time[] to timestamptz[]")
				}
			},
		},
	}

	for i, tt := range tests {
		err := conn.QueryRow(tt.sql, tt.query).Scan(tt.scan)
		if err != nil {
			t.Errorf(`%d. error reading array: %v`, i, err)
		}
		tt.assert(t, tt.query, tt.scan)
		ensureConnValid(t, conn)
	}
}

type shortScanner struct{}

func (*shortScanner) Scan(r *pgx.ValueReader) error {
	r.ReadByte()
	return nil
}

func TestShortScanner(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	rows, err := conn.Query("select 'ab', 'cd' union select 'cd', 'ef'")
	if err != nil {
		t.Error(err)
	}
	defer rows.Close()

	for rows.Next() {
		var s1, s2 shortScanner
		err = rows.Scan(&s1, &s2)
		if err != nil {
			t.Error(err)
		}
	}

	ensureConnValid(t, conn)
}

func TestEmptyArrayDecoding(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	var val []string

	err := conn.QueryRow("select array[]::text[]").Scan(&val)
	if err != nil {
		t.Errorf(`error reading array: %v`, err)
	}
	if len(val) != 0 {
		t.Errorf("Expected 0 values, got %d", len(val))
	}

	var n, m int32

	err = conn.QueryRow("select 1::integer, array[]::text[], 42::integer").Scan(&n, &val, &m)
	if err != nil {
		t.Errorf(`error reading array: %v`, err)
	}
	if len(val) != 0 {
		t.Errorf("Expected 0 values, got %d", len(val))
	}
	if n != 1 {
		t.Errorf("Expected n to be 1, but it was %d", n)
	}
	if m != 42 {
		t.Errorf("Expected n to be 42, but it was %d", n)
	}

	rows, err := conn.Query("select 1::integer, array['test']::text[] union select 2::integer, array[]::text[] union select 3::integer, array['test']::text[]")
	if err != nil {
		t.Errorf(`error retrieving rows with array: %v`, err)
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&n, &val)
		if err != nil {
			t.Errorf(`error reading array: %v`, err)
		}
	}

	ensureConnValid(t, conn)
}

func TestNullXMismatch(t *testing.T) {
	t.Parallel()

	conn := mustConnect(t, *defaultConnConfig)
	defer closeConn(t, conn)

	type allTypes struct {
		s   pgx.NullString
		i16 pgx.NullInt16
		i32 pgx.NullInt32
		i64 pgx.NullInt64
		f32 pgx.NullFloat32
		f64 pgx.NullFloat64
		b   pgx.NullBool
		t   pgx.NullTime
	}

	var actual, zero allTypes

	tests := []struct {
		sql       string
		queryArgs []interface{}
		scanArgs  []interface{}
		err       string
	}{
		{"select $1::date", []interface{}{pgx.NullString{String: "foo", Valid: true}}, []interface{}{&actual.s}, "invalid input syntax for type date"},
		{"select $1::date", []interface{}{pgx.NullInt16{Int16: 1, Valid: true}}, []interface{}{&actual.i16}, "cannot encode into OID 1082"},
		{"select $1::date", []interface{}{pgx.NullInt32{Int32: 1, Valid: true}}, []interface{}{&actual.i32}, "cannot encode into OID 1082"},
		{"select $1::date", []interface{}{pgx.NullInt64{Int64: 1, Valid: true}}, []interface{}{&actual.i64}, "cannot encode into OID 1082"},
		{"select $1::date", []interface{}{pgx.NullFloat32{Float32: 1.23, Valid: true}}, []interface{}{&actual.f32}, "cannot encode into OID 1082"},
		{"select $1::date", []interface{}{pgx.NullFloat64{Float64: 1.23, Valid: true}}, []interface{}{&actual.f64}, "cannot encode into OID 1082"},
		{"select $1::date", []interface{}{pgx.NullBool{Bool: true, Valid: true}}, []interface{}{&actual.b}, "cannot encode into OID 1082"},
		{"select $1::date", []interface{}{pgx.NullTime{Time: time.Unix(123, 5000), Valid: true}}, []interface{}{&actual.t}, "cannot encode into OID 1082"},
	}

	for i, tt := range tests {
		actual = zero

		err := conn.QueryRow(tt.sql, tt.queryArgs...).Scan(tt.scanArgs...)
		if err == nil || !strings.Contains(err.Error(), tt.err) {
			t.Errorf(`%d. Expected error to contain "%s", but it didn't: %v`, i, tt.err, err)
		}

		ensureConnValid(t, conn)
	}
}
