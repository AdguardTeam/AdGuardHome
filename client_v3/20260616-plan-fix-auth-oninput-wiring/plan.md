# Fix: Missing `onInput` handler in Auth.tsx causes "Please fill out this field" error on blur despite filled field

**Date**: 2026-06-16  
**Status**: 📋 Planned

---

## Summary

When the user types a value into the **username** field on the install page (step 2 — Auth form) and the input loses focus, it shows the error "Please fill out this field" even though text has been entered. This is a wiring issue: the `Field` render prop's `onInput` handler is not passed to the `<Input>` component, so the typed value is never stored in the form.

---

## Root Cause

**File**: `src/install/Setup/Auth.tsx`

In `@modular-forms/solid` v0.25.1, the `Field` render prop provides a second argument (`props`) with three event handlers:

| Handler | What it does |
|---------|-------------|
| `onInput` | **Extracts** the new value from `event.target`, **updates** `field.value`, marks touched, validates |
| `onChange` | Validates against **existing** `field.value` — does **NOT** extract new value, does **NOT** update value |
| `onBlur` | Validates against **existing** `field.value` — does **NOT** extract new value, marks touched |

**Only `onInput` extracts the typed value from the DOM element.** If `onInput` is not wired to the native `<input>`, the field value stays at its initial value (empty string `""`) regardless of what the user types.

### The current code (Auth.tsx)

The `<Field>` render prop children manually pass `onChange` and `onBlur` but **omit `onInput`**, `ref`, and `autofocus`:

```tsx
// Auth.tsx — username Field (lines ~108-122)
<Field name="username" validate={validateRequiredValue}>
    {(field, props) => (
        <Input
            type="text"
            id="install_username"
            label={...}
            placeholder={...}
            autocomplete="username"
            value={(field.value as string) ?? ''}
            onChange={props.onChange as (e: Event) => void}       // ← won't update value
            onBlur={props.onBlur as (e: FocusEvent) => void}      // ← validates against old value
            name={field.name}
            errorMessage={field.error as string}
            /* ❌ MISSING: onInput={props.onInput} */
            /* ❌ MISSING: ref={props.ref} */
            /* ❌ MISSING: autofocus={props.autofocus} */
        />
    )}
</Field>
```

### The sequence that causes the bug

| Step | User action | Native event | Handler called | Field value after | 
|------|------------|-------------|----------------|-------------------|
| 1 | Types "j" | `input` | ❌ None | `""` (unchanged) |
| 2 | Types "o" | `input` | ❌ None | `""` (unchanged) |
| 3 | Types "h" | `input` | ❌ None | `""` (unchanged) |
| 4 | Types "n" | `input` | ❌ None | `""` (unchanged) |
| 5 | Tabs away | `change` | `props.onChange` → validates against `""` | `""` |
| 6 | Tabs away | `blur` | `props.onBlur` → validates against `""` | `""` → **validateRequiredValue("") → "Please fill out this field"** |

The `Input` component's native `<input>` fires `onInput` on every keystroke, but `props.onInput` is `undefined`, so no handler runs. When blur fires, it validates against the empty `""` value and shows the error.

### Why the password fields don't have the same problem

The password `<PasswordInput>` fields bypass the Field's value extraction entirely by calling `setValue()` directly in `onChange`:

```tsx
// This works because setValue() explicitly updates the form value
onChange={(value) => setValue(authForm, 'password', value, { shouldValidate: true })}
```

However, they still miss `ref` and `onBlur` wiring from the Field.

### The correct pattern (reference: `src/login/Login/Form.tsx`)

The working login form spreads `{...fieldProps}` onto the Input, which includes `onInput`, `ref`, `autofocus`, `name`, `onChange`, and `onBlur`:

```tsx
// Login Form.tsx — username Field
<Field name="username" validate={[required(...)]}>
    {(field, fieldProps) => (
        <Input
            {...fieldProps}               // ✅ Spreads ALL handlers
            id="username"
            type="text"
            value={(field.value as string) || ''}
            label={...}
            placeholder={...}
            errorMessage={field.error as string}
            autocomplete="username"
            autocapitalize="none"
        />
    )}
</Field>
```

The spread happens **before** the explicit overrides, so `value`, `errorMessage`, etc. override anything in the fieldProps (though fieldProps doesn't include those).

---

## Fix

### File to modify

`src/install/Setup/Auth.tsx`

### Strategy

For each `<Field>`, spread `{...props}` (the second Field render prop argument) onto the native control **before** the explicit overrides. This wires `onInput`, `ref`, `autofocus`, and `name` from the Field without needing to list them individually.

#### 1. Fix username `<Input>` (lines ~108-122)

**Before:**
```tsx
{(field, props) => (
    <Input
        type="text"
        id="install_username"
        label={intl.getMessage('install_auth_username')}
        placeholder={intl.getMessage('install_auth_username_enter')}
        autocomplete="username"
        value={(field.value as string) ?? ''}
        onChange={props.onChange as (e: Event) => void}
        onBlur={props.onBlur as (e: FocusEvent) => void}
        name={field.name}
        errorMessage={field.error as string}
    />
)}
```

**After:**
```tsx
{(field, props) => (
    <Input
        {...props}
        type="text"
        id="install_username"
        label={intl.getMessage('install_auth_username')}
        placeholder={intl.getMessage('install_auth_username_enter')}
        autocomplete="username"
        value={(field.value as string) ?? ''}
        errorMessage={field.error as string}
    />
)}
```

**What changes:**
- Added `{...props}` spread — wires `onInput`, `onChange`, `onBlur`, `ref`, `autofocus`, `name`
- Removed manual `onChange={...}`, `onBlur={...}`, `name={field.name}` — now handled by spread

#### 2. Fix password `<PasswordInput>` (lines ~127-142)

**Before:**
```tsx
{(field, props) => (
    <PasswordInput
        id="install_password"
        label={intl.getMessage('install_auth_password')}
        placeholder={intl.getMessage('install_auth_password_enter')}
        autocomplete="new-password"
        value={(field.value as string) ?? ''}
        onChange={(value) => setValue(authForm, 'password', value, { shouldValidate: true })}
        name={field.name}
        onBlur={props.onBlur as (e: FocusEvent) => void}
        errorMessage={field.error as string}
    />
)}
```

**After:**
```tsx
{(field, props) => (
    <PasswordInput
        {...props}
        id="install_password"
        label={intl.getMessage('install_auth_password')}
        placeholder={intl.getMessage('install_auth_password_enter')}
        autocomplete="new-password"
        value={(field.value as string) ?? ''}
        onChange={(value) => setValue(authForm, 'password', value, { shouldValidate: true })}
        errorMessage={field.error as string}
    />
)}
```

**What changes:**
- Added `{...props}` spread — wires `ref`, `onBlur`, `autofocus`, `name`
- Removed manual `name={field.name}`, `onBlur={props.onBlur ...}` — now handled by spread
- `onChange` override is kept (and placed after the spread) because PasswordInput has different `onChange` semantics
- `PasswordInput` internally calls `value` → `setValue` in its `onChange`, so the `onChange` from the spread gets overridden

#### 3. Fix confirm_password `<PasswordInput>` (lines ~161-176)

**Before:**
```tsx
{(field, props) => (
    <PasswordInput
        id="install_confirm_password"
        label={intl.getMessage('install_auth_confirm')}
        placeholder={intl.getMessage('install_auth_confirm')}
        autocomplete="new-password"
        value={(field.value as string) ?? ''}
        onChange={(value) => setValue(authForm, 'confirm_password', value, { shouldValidate: true })}
        name={field.name}
        onBlur={props.onBlur as (e: FocusEvent) => void}
        errorMessage={field.error as string}
    />
)}
```

**After:**
```tsx
{(field, props) => (
    <PasswordInput
        {...props}
        id="install_confirm_password"
        label={intl.getMessage('install_auth_confirm')}
        placeholder={intl.getMessage('install_auth_confirm')}
        autocomplete="new-password"
        value={(field.value as string) ?? ''}
        onChange={(value) => setValue(authForm, 'confirm_password', value, { shouldValidate: true })}
        errorMessage={field.error as string}
    />
)}
```

**What changes:** Same as password field above.

#### 4. Fix privacy_consent `<Checkbox>` (lines ~183-226)

**Before:**
```tsx
{(field, props) => (
    <Checkbox
        checked={(field.value as boolean) ?? false}
        onChange={(e: Event) => setValue(authForm, 'privacy_consent', (e.target as HTMLInputElement).checked, { shouldValidate: true })}
        name={field.name}
        onBlur={props.onBlur as (e: FocusEvent) => void}
        verticalAlign="start"
    >
```

**After:**
```tsx
{(field, props) => (
    <Checkbox
        {...props}
        checked={(field.value as boolean) ?? false}
        onChange={(e: Event) => setValue(authForm, 'privacy_consent', (e.target as HTMLInputElement).checked, { shouldValidate: true })}
        verticalAlign="start"
    >
```

**What changes:**
- Added `{...props}` spread — wires `ref`, `onBlur`, `autofocus`, `name`
- Removed manual `name={field.name}`, `onBlur={props.onBlur ...}` — now handled by spread
- `onChange` override is kept (and placed after the spread) because it uses `setValue` for boolean handling

---

## Files Changed

| File | Change |
|------|--------|
| `src/install/Setup/Auth.tsx` | Add `{...props}` spread to all 4 Field render props; remove redundant manual prop wiring |

---

## Verification Checklist

- [ ] `npm run dev` — dev server starts without errors (webpack)
- [ ] Navigate to the installation page (`/install.html`)
- [ ] Press "Run wizard" button — page transitions to step 2 (Auth form) without errors
- [ ] **Type in username field** — text appears, no "please fill" error on blur
- [ ] Clear username and blur empty — "Please fill out this field" error shows
- [ ] Fill in password — requirements checklist updates in real time
- [ ] Fill in confirm password — match validation works
- [ ] Check privacy consent checkbox — privacy consent validates
- [ ] Click "Next" — advances to step 3 (Interface Settings)
- [ ] Click "Back" from step 3 — returns to step 2 with form values preserved
- [ ] Run E2E tests: `npx playwright test tests/e2e/` — no regressions

---

## Risk Assessment

- **Risk level**: Very low
- **Affected area**: Installation wizard step 2 only (Auth form)
- **Backward compatibility**: 100% — the change only wires previously-missing Field handlers; validation logic and form behavior are unchanged
- **Rollback**: Trivial (revert the 4 Field render props to previous wiring)
