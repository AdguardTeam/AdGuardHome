 #  AdGuard Home Developer Guidelines

Following this document is obligatory for all new code.  Some of the rules
aren't enforced as thoroughly or remain broken in old code, but this is still
the place to find out about what we **want** our code to look like and how to
improve it.

The rules are mostly sorted in the alphabetical order.



## Contents

 *  [Git](#git)
 *  [Go](#go)
     *  [Code](#code)
     *  [Commenting](#commenting)
     *  [Formatting](#formatting)
     *  [Naming](#naming)
     *  [Testing](#testing)
     *  [Recommended Reading](#recommended-reading)
 *  [Markdown](#markdown)
 *  [Shell Scripting](#shell-scripting)
     *  [Shell Conditionals](#shell-conditionals)
 *  [Text, Including Comments](#text-including-comments)
 *  [YAML](#yaml)

<!-- NOTE: Use the IDs that GitHub would generate in order for this to work both
on GitHub and most other Markdown renderers.  Use both "id" and "name"
attributes to make it work in Markdown renderers that strip "id". -->



##  <a href="#git" id="git" name="git">Git</a>

 *  Call your branches either `NNNN-fix-foo` (where `NNNN` is the ID of the
    GitHub issue you worked on in this branch) or just `fix-foo` if there was no
    GitHub issue.

 *  Follow the commit message header format:

    ```none
    pkg: fix the network error logging issue
    ```

    Where `pkg` is the directory or Go package (without the `internal/` part)
    where most changes took place.  If there are several such packages, or the
    change is top-level only, write `all`.

 *  Keep your commit messages, including headers, to eighty (**80**) columns.

 *  Only use lowercase letters in your commit message headers.  The rest of the
    message should follow the plain text conventions below.

    The only exceptions are direct mentions of identifiers from the source code
    and filenames like `HACKING.md`.



##  <a href="#go" id="go" name="go">Go</a>

> Not Golang, not GO, not GOLANG, not GoLang. It is Go in natural language,
> golang for others.

— [@rakyll](https://twitter.com/rakyll/status/1229850223184269312)

 ###  <a href="#code" id="code" name="code">Code</a>

 *  Always `recover` from panics in new goroutines.  Preferably in the very
    first statement.  If all you want there is a log message, use `log.OnPanic`.

 *  Avoid `fallthrough`.  It makes it harder to rearrange `case`s, to reason
    about the code, and also to switch the code to a handler approach, if that
    becomes necessary later.

 *  Avoid `goto`.

 *  Avoid `init` and use explicit initialization functions instead.

 *  Avoid `new`, especially with structs, unless a temporary value is needed,
    for example when checking the type of an error using `errors.As`.

 *  Check against empty strings like this:

    ```go
    if s == "" {
            // …
    }
    ```

    Except when the check is done to then use the first character:

    ```go
    if len(s) > 0 {
            c := s[0]
    }
    ```

 *  Constructors should validate their arguments and return meaningful errors.
    As a corollary, avoid lazy initialization.

 *  Prefer to define methods on pointer receievers, unless the type is small or
    a non-pointer receiever is required, for example `MarshalFoo` methods (see
    [staticcheck-911]).

 *  Don't mix horizontal and vertical placement of arguments in function and
    method calls.  That is, either this:

    ```go
    err := f(a, b, c)
    ```

    Or, when the arguments are too long, this:

    ```go
    err := functionWithALongName(
            firstArgumentWithALongName,
            secondArgumentWithALongName,
            thirdArgumentWithALongName,
    )
    ```

    Or, with a struct literal:

    ```go
    err := functionWithALongName(arg, structType{
            field1: val1,
            field2: val2,
    })
    ```

    But **never** this:

    ```go
    err := functionWithALongName(firstArgumentWithALongName,
            secondArgumentWithALongName,
            thirdArgumentWithALongName,
    )
    ```

 *  Don't rely only on file names for build tags to work.  Always add build tags
    as well.

 *  Don't use `fmt.Sprintf` where a more structured approach to string
    conversion could be used.  For example, `aghnet.JoinHostPort`,
    `net.JoinHostPort` or `url.(*URL).String`.

 *  Don't use naked `return`s.

 *  Don't write non-test code with more than four (**4**) levels of indentation.
    Just like [Linus said], plus an additional level for an occasional error
    check or struct initialization.

    The exception proving the rule is the table-driven test code, where an
    additional level of indentation is allowed.

 *  Eschew external dependencies, including transitive, unless absolutely
    necessary.

 *  Minimize scope of variables as much as possible.

 *  No name shadowing, including of predeclared identifiers, since it can often
    lead to subtle bugs, especially with errors.  This rule does not apply to
    struct fields, since they are always used together with the name of the
    struct value, so there isn't any confusion.

 *  Prefer constants to variables where possible.  Avoid global variables.  Use
    [constant errors] instead of `errors.New`.

 *  Prefer defining `Foo.String` and `ParseFoo` in terms of `Foo.MarshalText`
    and `Foo.UnmarshalText` correspondingly and not the other way around.

 *  Prefer to use named functions for goroutines.

 *  Program code lines should not be longer than one hundred (**100**) columns.
    For comments, see the text section below.

 *  Use linters.  `make go-lint`.

 *  Write logs and error messages in lowercase only to make it easier to `grep`
    logs and error messages without using the `-i` flag.

 ###  <a href="#commenting" id="commenting" name="commenting">Commenting</a>

 *  See also the “[Text, Including Comments]” section below.

 *  Document everything, including unexported top-level identifiers, to build
    a habit of writing documentation.

 *  Don't put identifiers into any kind of quotes.

 *  Put comments above the documented entity, **not** to the side, to improve
    readability.

 *  When a method implements an interface, start the doc comment with the
    standard template:

    ```go
    // Foo implements the Fooer interface for *foo.
    func (f *foo) Foo() {
            // …
    }
    ```

    When the implemented interface is unexported:

    ```go
    // Unwrap implements the hidden wrapper interface for *fooError.
    func (err *fooError) Unwrap() (unwrapped error) {
            // …
    }
    ```

 ###  <a href="#formatting" id="formatting" name="formatting">Formatting</a>

 *  Decorate `break`, `continue`, `return`, and other terminating statements
    with empty lines unless it's the only statement in that block.

 *  Don't group type declarations together.  Unlike with blocks of `const`s,
    where a `iota` may be used or where all constants belong to a certain type,
    there is no reason to group `type`s.

 *  Group `require.*` blocks together with the presceding related statements,
    but separate from the following `assert.*` and unrelated requirements.

    ```go
    val, ok := valMap[key]
    require.True(t, ok)
    require.NotNil(t, val)

    assert.Equal(t, expected, val)
    ```

 *  Use `gofumpt --extra -s`.

 *  Write slices of struct like this:

    ```go
    ts := []T{{
            Field: Value0,
            // …
    }, {
            Field: Value1,
            // …
    }, {
            Field: Value2,
            // …
    }}
    ```

 ###  <a href="#naming" id="naming" name="naming">Naming</a>

 *  Don't use underscores in file and package names, unless they're build tags
    or for tests.  This is to prevent accidental build errors with weird tags.

 *  Name benchmarks and tests using the same convention as examples.  For
    example:

    ```go
    func TestFunction(t *testing.T) { /* … */ }
    func TestFunction_suffix(t *testing.T) { /* … */ }
    func TestType_Method(t *testing.T) { /* … */ }
    func TestType_Method_suffix(t *testing.T) { /* … */ }
    ```

 *  Name parameters in interface definitions:

    ```go
    type Frobulator interface {
            Frobulate(f Foo, b Bar) (r Result, err error)
    }
    ```

 *  Name the deferred errors (e.g. when closing something) `derr`.

 *  Unused arguments in anonymous functions must be called `_`:

    ```go
    v.onSuccess = func(_ int, msg string) {
            // …
    }
    ```

 *  Use named returns to improve readability of function signatures.

 *  When naming a file which defines an entity, use singular nouns, unless the
    entity is some form of a container for other entities:

    ```go
    // File: client.go

    package foo

    type Client struct {
            // …
    }
    ```

    ```go
    // File: clients.go

    package foo

    type Clients []*Client

    // …

    type ClientsWithCache struct {
            // …
    }
    ```

 ###  <a href="#testing" id="testing" name="testing">Testing</a>

 *  Use `assert.NoError` and `require.NoError` instead of `assert.Nil` and
    `require.Nil` on errors.

 *  Use formatted helpers, like `assert.Nilf` or `require.Nilf`, instead of
    simple helpers when a formatted message is required.

 *  Use functions like `require.Foo` instead of `assert.Foo` when the test
    cannot continue if the condition is false.

 ###  <a href="#recommended-reading" id="recommended-reading" name="recommended-reading">Recommended Reading</a>

 *  <https://github.com/golang/go/wiki/CodeReviewComments>.

 *  <https://github.com/golang/go/wiki/TestComments>.

 *  <https://go-proverbs.github.io/>

[Linus said]:               https://www.kernel.org/doc/html/v4.17/process/coding-style.html#indentation
[Text, Including Comments]: #text-including-comments
[constant errors]:          https://dave.cheney.net/2016/04/07/constant-errors
[staticcheck-911]:          https://github.com/dominikh/go-tools/issues/911



##  <a href="#markdown" id="markdown" name="markdown">Markdown</a>

 *  **TODO(a.garipov):** Define more Markdown conventions.

 *  Prefer triple-backtick preformatted code blocks to indented code blocks.

 *  Use asterisks and not underscores for bold and italic.

 *  Use either link references or link destinations only.  Put all link
    reference definitions at the end of the second-level block.



##  <a href="#shell-scripting" id="shell-scripting" name="shell-scripting">Shell Scripting</a>

 *  Avoid bashisms and GNUisms, prefer POSIX features only.

 *  Avoid spaces between patterns of the same `case` condition.

 *  `export` and `readonly` should be used separately from variable assignment,
    because otherwise failures in command substitutions won't stop the script.
    That is, do this:

    ```sh
    X="$( echo 42 )"
    export X
    ```

    And **not** this:

    ```sh
    # Bad!
    export X="$( echo 42 )"
    ```

 *  If a binary value is needed, use `0` for `false`, and `1` for `true`.

 *  Mark every variable that shouldn't change later as `readonly`.

 *  Prefer `'raw strings'` to `"double quoted strings"` whenever possible.

 *  Put spaces within `$( cmd )`, `$(( expr ))`, and `{ cmd; }`.

 *  Put utility flags in the ASCII order and **don't** group them together.  For
    example, `ls -1 -A -q`.

 *  Script code lines should not be longer than one hundred (**100**) columns.
    For comments, see the text section below.

 *  `snake_case`, not `camelCase` for variables.  `kebab-case` for filenames.

 *  Start scripts with the following sections in the following order:

    1.  Shebang.
    1.  Some initial documentation (optional).
    1.  Verbosity level parsing (optional).
    1.  `set` options.

 *  UPPERCASE names for external exported variables, lowercase for local,
    unexported ones.

 *  Use `set -e -f -u` and also `set -x` in verbose mode.

 *  Use the `"$var"` form instead of the `$var` form, unless word splitting is
    required.

 *  When concatenating, always use the form with curly braces to prevent
    accidental bad variable names.  That is, `"${var}_tmp.txt"` and **not**
    `"$var_tmp.txt"`.  The latter will try to lookup variable `var_tmp`.

 *  When concatenating, surround the whole string with quotes.  That is, use
    this:

    ```sh
    dir="${TOP_DIR}/sub"
    ```

    And **not** this:

    ```sh
    # Bad!
    dir="${TOP_DIR}"/sub
    ```

 ###  <a href="#shell-conditionals" id="shell-conditionals" name="shell-conditionals">Shell Conditionals</a>

Guidelines and agreements for using command `test`, also known as `[`:

 *  For conditionals that check for equality against multiple values, prefer
    `case` instead of `test`.

 *  Prefer the `!= ''` form instead of using `-n` to check if string is empty.

 *  Spell compound conditions with `&&`, `||`, and `!` **outside** of `test`
    instead of `-a`, `-o`, and `!` **inside** of `test` correspondingly.  The
    latter ones are pretty much deprecated in POSIX.

    See also: “[Problems With the `test` Builtin: What Does `-a` Mean?][test]”.

 *  Use `=` for strings and `-eq` for numbers to catch typing errors.

[test]: https://www.oilshell.org/blog/2017/08/31.html



##  <a href="#text-including-comments" id="text-including-comments" name="text-including-comments">Text, Including Comments</a>

 *  End sentences with appropriate punctuation.

 *  Headers should be written with all initial letters capitalized, except for
    references to variable names that start with a lowercase letter.

 *  Mark temporary todos—that is, todos that must be resolved or removed before
    publishing a change—with two exclamation signs:

    ```go
    // TODO(usr1): !! Remove this debug before pushing!
    ```

    This makes it easier to find them both during development and during code
    review.

 *  Start sentences with a capital letter, unless the first word is a reference
    to a variable name that starts with a lowercase letter.

 *  Text should wrap at eighty (**80**) columns to be more readable, to use
    a common standard, and to allow editing or diffing side-by-side without
    wrapping.

    The only exception are long hyperlinks.

 *  Use U.S. English, as it is the most widely used variety of English in the
    code right now as well as generally.

 *  Use double spacing between sentences to make sentence borders more clear.

 *  Use the serial comma (a.k.a. Oxford comma) to improve comprehension,
    decrease ambiguity, and use a common standard.

 *  Write todos like this:

    ```go
    // TODO(usr1): Fix the frobulation issue.
    ```

    Or, if several people need to look at the code:

    ```go
    // TODO(usr1, usr2): Fix the frobulation issue.
    ```



##  <a href="#yaml" id="yaml" name="yaml">YAML</a>

 *  **TODO(a.garipov):** Define naming conventions for schema names in our
    OpenAPI YAML file.  And just generally OpenAPI conventions.

 *  **TODO(a.garipov):** Find a YAML formatter or write our own.

 *  All strings, including keys, must be quoted.  Reason: the “[NO-rway Law]”.

 *  Indent with two (**2**) spaces.  YAML documents can get pretty
    deeply-nested.

 *  No extra indentation in multiline arrays:

    ```yaml
    'values':
    - 'value-1'
    - 'value-2'
    - 'value-3'
    ```

 *  Prefer single quotes for strings to prevent accidental escaping, unless
    escaping is required or there are single quotes inside the string (e.g. for
    GitHub Actions).

 *  Use `>` for multiline strings, unless you need to keep the line breaks.  Use
    `|` for multiline strings when you do.

[NO-rway Law]: https://news.ycombinator.com/item?id=17359376
