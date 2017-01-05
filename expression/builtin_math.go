// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package expression

import (
	"hash/crc32"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/juju/errors"
	"github.com/pingcap/tidb/context"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/util/types"
)

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_abs
func builtinAbs(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	d = args[0]
	switch d.Kind() {
	case types.KindNull:
		return d, nil
	case types.KindUint64:
		return d, nil
	case types.KindInt64:
		iv := d.GetInt64()
		if iv >= 0 {
			d.SetInt64(iv)
			return d, nil
		}
		d.SetInt64(-iv)
		return d, nil
	default:
		// we will try to convert other types to float
		// TODO: if time has no precision, it will be a integer
		f, err := d.ToFloat64(ctx.GetSessionVars().StmtCtx)
		d.SetFloat64(math.Abs(f))
		return d, errors.Trace(err)
	}
}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_ceiling
func builtinCeil(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	if args[0].IsNull() ||
		args[0].Kind() == types.KindUint64 || args[0].Kind() == types.KindInt64 {
		return args[0], nil
	}

	f, err := args[0].ToFloat64(ctx.GetSessionVars().StmtCtx)
	if err != nil {
		return d, errors.Trace(err)
	}
	d.SetFloat64(math.Ceil(f))
	return
}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_log
func builtinLog(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	sc := ctx.GetSessionVars().StmtCtx

	switch len(args) {
	case 1:
		x, err := args[0].ToFloat64(sc)
		if err != nil {
			return d, errors.Trace(err)
		}

		if x <= 0 {
			return d, nil
		}

		d.SetFloat64(math.Log(x))
		return d, nil
	case 2:
		b, err := args[0].ToFloat64(sc)
		if err != nil {
			return d, errors.Trace(err)
		}

		x, err := args[1].ToFloat64(sc)
		if err != nil {
			return d, errors.Trace(err)
		}

		if b <= 1 || x <= 0 {
			return d, nil
		}

		d.SetFloat64(math.Log(x) / math.Log(b))
		return d, nil
	}
	return
}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_log2
func builtinLog2(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	sc := ctx.GetSessionVars().StmtCtx
	x, err := args[0].ToFloat64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	if x <= 0 {
		return
	}

	d.SetFloat64(math.Log2(x))
	return
}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_log10
func builtinLog10(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	sc := ctx.GetSessionVars().StmtCtx
	x, err := args[0].ToFloat64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	if x <= 0 {
		return
	}

	d.SetFloat64(math.Log10(x))
	return

}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_rand
func builtinRand(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	if len(args) == 1 && !args[0].IsNull() {
		seed, err := args[0].ToInt64(ctx.GetSessionVars().StmtCtx)
		if err != nil {
			return d, errors.Trace(err)
		}
		rand.Seed(seed)
	}
	d.SetFloat64(rand.Float64())
	return d, nil
}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_pow
func builtinPow(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	sc := ctx.GetSessionVars().StmtCtx
	x, err := args[0].ToFloat64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	y, err := args[1].ToFloat64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}
	d.SetFloat64(math.Pow(x, y))
	return d, nil
}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_round
func builtinRound(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	sc := ctx.GetSessionVars().StmtCtx
	x, err := args[0].ToFloat64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	dec := 0
	if len(args) == 2 {
		y, err1 := args[1].ToInt64(sc)
		if err1 != nil {
			return d, errors.Trace(err1)
		}
		dec = int(y)
	}
	d.SetFloat64(types.Round(x, dec))
	return d, nil
}

// See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_conv
func builtinConv(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	var (
		n        string
		toBase   int64
		fromBase int64
		signed   bool
		negative bool
		touval   bool
	)
	for _, arg := range args {
		if arg.IsNull() {
			return d, nil
		}
	}
	n, err = args[0].ToString()
	if err != nil {
		return d, errors.Trace(err)
	}
	sc := ctx.GetSessionVars().StmtCtx
	fromBase, err = args[1].ToInt64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}
	toBase, err = args[2].ToInt64(sc)
	if err != nil {
		return d, errors.Trace(err)
	}

	if fromBase < 0 {
		fromBase = -fromBase
		signed = true
	}
	if toBase < 0 {
		touval = true
		toBase = -toBase
	}
	if fromBase > 36 || fromBase < 2 || toBase > 36 || toBase < 2 {
		return d, nil
	}
	n = getValidPrefix(n, fromBase)
	if len(n) == 0 {
		return d, nil
	}
	if n[0] == '-' {
		negative = true
		n = n[1:]
	}

	val, err := strconv.ParseUint(n, int(fromBase), 64)
	if err != nil {
		return d, errors.Trace(types.ErrOverflow)
	}
	//Ref https://github.com/mysql/mysql-server/blob/5.7/strings/ctype-simple.c#L598
	if signed {
		if negative && val > -math.MinInt64 {
			val = -math.MinInt64
		}
		if !negative && val > math.MaxInt64 {
			val = math.MaxInt64
		}
	}
	if negative {
		val = -val
	}
	// Ref https://github.com/mysql/mysql-server/blob/5.7/strings/longlong2str.c#L58
	if int64(val) < 0 {
		negative = true
	} else {
		negative = false
	}
	if touval && negative {
		val = 0 - val
	}

	s := strconv.FormatUint(val, int(toBase))
	if negative && touval {
		s = "-" + s
	}
	d.SetString(strings.ToUpper(s))
	return d, nil
}

//　See http://dev.mysql.com/doc/refman/5.7/en/mathematical-functions.html#function_crc32
func builtinCRC32(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
	if args[0].IsNull() {
		return d, nil
	}
	x, err := args[0].ToString()
	if err != nil {
		return d, errors.Trace(err)
	}
	r := crc32.ChecksumIEEE([]byte(x))
	d.SetUint64(uint64(r))
	return d, nil
}

func arithmeticFuncFactory(op opcode.Op) BuiltinFunc {
	return func(args []types.Datum, ctx context.Context) (d types.Datum, err error) {
		sc := ctx.GetSessionVars().StmtCtx
		a, err := types.CoerceArithmetic(sc, args[0])
		if err != nil {
			return d, errors.Trace(err)
		}

		b, err := types.CoerceArithmetic(sc, args[1])
		if err != nil {
			return d, errors.Trace(err)
		}
		a, b, err = types.CoerceDatum(sc, a, b)
		if err != nil {
			return d, errors.Trace(err)
		}
		if a.IsNull() || b.IsNull() {
			return
		}

		switch op {
		case opcode.Plus:
			return types.ComputePlus(a, b)
		case opcode.Minus:
			return types.ComputeMinus(a, b)
		case opcode.Mul:
			return types.ComputeMul(a, b)
		case opcode.Div:
			return types.ComputeDiv(sc, a, b)
		case opcode.Mod:
			return types.ComputeMod(sc, a, b)
		case opcode.IntDiv:
			return types.ComputeIntDiv(sc, a, b)
		default:
			return d, errInvalidOperation.Gen("invalid op %v in arithmetic operation", op)
		}
	}
}
