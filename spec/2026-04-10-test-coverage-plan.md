# Plan: Fill Test Coverage Gaps

## Context

Grove has strong integration-level coverage (88.3% in pkg/grove) but zero coverage in the
internal packages (vm, compiler, parser, filters, scope). Key gaps: sandbox enforcement
is entirely untested, filter error paths are weak, component edge cases are missing, and
several VM/parser error paths have no tests.

## Approach

All tests go in `pkg/grove/` using the existing helper pattern:
- `render(t, eng, tmpl, data)` — inline, expect success
- `renderErr(t, eng, tmpl, data)` — inline, return error for assertion
- `renderStore/renderComponent` — multi-template tests with MemoryStore
- Error assertions use `require.ErrorAs` for typed errors, `require.Contains(err.Error(), ...)` for casual checks

## Files to Create / Extend

### 1. `pkg/grove/sandbox_test.go` — NEW
Tests for AllowedTags, AllowedFilters, MaxLoopIter enforcement. Currently 0% tested.

```
TestSandbox_AllowedTags_BlocksIf
TestSandbox_AllowedTags_BlocksEach
TestSandbox_AllowedTags_BlocksImport
TestSandbox_AllowedTags_AllowsSet
TestSandbox_AllowedTags_NilAllowsAll
TestSandbox_AllowedFilters_BlocksUpper
TestSandbox_AllowedFilters_AllowsWhitelisted
TestSandbox_AllowedFilters_NilAllowsAll
TestSandbox_MaxLoopIter_Exceeded
TestSandbox_MaxLoopIter_NotExceeded
TestSandbox_MaxLoopIter_ZeroIsUnlimited
TestSandbox_MaxLoopIter_NestedLoops
```

### 2. `pkg/grove/errors_test.go` — NEW
VM runtime error paths, type errors, coercion failures.

```
TestError_ModuloByZero
TestError_IndexOnNonCollection
TestError_IndexOutOfBounds
TestError_AttrOnNil
TestError_MissingRequiredProp
TestError_StrictMode_NestedMissing
TestError_ParseError_UnclosedIf
TestError_ParseError_UnclosedEach
TestError_ParseError_UnclosedLet
TestError_ParseError_UnclosedCapture
TestError_ParseError_UnclosedFill
TestError_ParseError_UnclosedSlot
TestError_ParseError_UnclosedVerbatim
TestError_ParseError_UnclosedHoist
TestError_ParseError_InvalidSet
TestError_ParseError_InvalidImport
TestError_ParseError_LineNumbers
```

### 3. Additions to `pkg/grove/filters_test.go` — EXTEND
Edge cases for existing filters that have no boundary or nil coverage.

```
TestFilter_EdgeCases_NilInput       — nil through upper/lower/trim/etc
TestFilter_EdgeCases_EmptyString    — "" through all string filters
TestFilter_EdgeCases_EmptyList      — [] through first/last/join/sort/min/max
TestFilter_Truncate_LongerThanInput — truncate(n) where n > len
TestFilter_Truncate_ZeroLen         — truncate(0)
TestFilter_Batch_SizeOne            — batch(1)
TestFilter_Batch_LargerThanList     — batch(100) on 3-item list
TestFilter_Sort_AllSame             — sort list of identical values
TestFilter_Min_SingleItem           — min/max on single-element list
TestFilter_Max_SingleItem
TestFilter_Sum_MixedIntFloat        — sum of [1, 2.5, 3]
TestFilter_Map_MissingField         — map("nonexistent") returns list of nil
TestFilter_Flatten_AlreadyFlat      — flatten on non-nested list
TestFilter_Keys_EmptyMap
TestFilter_Values_EmptyMap
TestFilter_Default_ZeroInt          — default("x") where value is 0 (truthy since 0 is not nil/false/"")
TestFilter_Split_EmptySep           — split("")
TestFilter_Join_EmptyList
TestFilter_Replace_EmptyNewStr
```

### 4. Additions to `pkg/grove/controlflow_test.go` — EXTEND
Missing edge cases for if/each/let/capture.

```
TestEach_EmptyLiteral              — {% #each [] as x %} goes to :empty
TestEach_TwoVarOnList              — {% #each items as i, item %}
TestEach_TwoVarOnMap               — {% #each map as k, v %}
TestEach_NestedLoopParent          — loop.parent in nested each
TestEach_RangeNegativeStep         — range(10, 0, -2)
TestIf_NilIsFalsy                  — {% #if nil %}
TestIf_ZeroIsFalsy                 — {% #if 0 %}
TestIf_EmptyStringIsFalsy          — {% #if "" %}
TestIf_EmptyListIsFalsy            — {% #if [] %}
TestLet_MultipleAssignments        — multiple vars in let block
TestLet_ConditionalBranch          — let with :else if branching
TestCapture_FilterOnResult         — capture then pipe result through filter
TestCapture_NestedCapture          — capture inside another capture
TestSet_Overwrite                  — set same var twice
```

### 5. Additions to `pkg/grove/component_test.go` — EXTEND
Component edge cases not covered.

```
TestComponent_NoProps              — component with zero declared props
TestComponent_DefaultPropUsed      — default value when prop not passed
TestComponent_DefaultPropOverridden — default overridden by caller
TestComponent_FillNoMatchingSlot   — fill for slot that doesn't exist renders nothing
TestComponent_SlotWithDefaultContent — named slot fallback renders when no fill
TestComponent_NestedSlotInFill     — slot inside a fill inside a slot
TestComponent_EmptyBody            — component invoked with no children
```

## Execution Order

1. `sandbox_test.go` — new file, independent
2. `errors_test.go` — new file
3. `filters_test.go` additions
4. `controlflow_test.go` additions
5. `component_test.go` additions

## Verification

```bash
go clean -testcache && go test ./pkg/grove/... -v -run "TestSandbox|TestError|TestFilter_Edge|TestFilter_Batch|TestFilter_Truncate|TestComponent_No|TestEach_Two|TestLet_|TestCapture_"
go test ./... -cover
```

Target: pkg/grove coverage from 88.3% → 93%+
