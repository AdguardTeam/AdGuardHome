 #  AdGuard Home Developer Guidelines

As of **March 2021**, following this document is obligatory for all new code.
Some of the rules aren't enforced as thoroughly or remain broken in old code,
but this is still the place to find out about what we **want** our code to look
like and how to improve it.

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
 *  [Text, Including Comments](#text-including-comments)
 *  [YAML](#yaml)

<!-- NOTE: Use the IDs that GitHub would generate in order for this to work both
on GitHub and most other Markdown renderers. -->



##  <a id="git" href="#git">Git</a>

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



##  <a id="go" href="#go">Go</a>

> Not Golang, not GO, not GOLANG, not GoLang. It is Go in natural language,
> golang for others.

— [@rakyll](https://twitter.com/rakyll/status/1229850223184269312)

 ###  <a id="code" href="#code">Code</a>

 *  Always `recover` from panics in new goroutines.  Preferably in the very
    first statement.  If all you want there is a log message, use
    `agherr.LogPanic`.

 *  Avoid `errors.New`, use `aghnet.Error` instead.

 *  Avoid `goto`.

 *  Avoid `init` and use explicit initialization functions instead.

 *  Avoid `new`, especially with structs.

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

    But **never** this:

    ```go
    err := functionWithALongName(firstArgumentWithALongName,
            secondArgumentWithALongName,
            thirdArgumentWithALongName,
    )
    ```

 *  Don't use `fmt.Sprintf` where a more structured approach to string
    conversion could be used.  For example, `net.JoinHostPort` or
    `url.(*URL).String`.

 *  Don't use naked `return`s.

 *  Don't write non-test code with more than four (**4**) levels of indentation.
    Just like [Linus said], plus an additional level for an occasional error
    check or struct initialization.

    The exception proving the rule is the table-driven test code, where an
    additional level of indentation is allowed.

 *  Eschew external dependencies, including transitive, unless
    absolutely necessary.

 *  Minimize scope of variables as much as possible.

 *  No shadowing, since it can often lead to subtle bugs, especially with
    errors.

 *  Prefer constants to variables where possible.  Reduce global variables.  Use
    [constant errors] instead of `errors.New`.

 *  Prefer to use named functions for goroutines.

 *  Program code lines should not be longer than one hundred (**100**) columns.
    For comments, see the text section below.

 *  Use linters.  `make go-lint`.

 *  Write logs and error messages in lowercase only to make it easier to `grep`
    logs and error messages without using the `-i` flag.

 ###  <a id="commenting" href="#commenting">Commenting</a>

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

 ###  <a id="formatting" href="#formatting">Formatting</a>

 *  Decorate `break`, `continue`, `fallthrough`, `return`, and other function
    exit points with empty lines unless it's the only statement in that block.

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

 ###  <a id="naming" href="#naming">Naming</a>

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

 ###  <a id="testing" href="#testing">Testing</a>

 *  Use `assert.NoError` and `require.NoError` instead of `assert.Nil` and
    `require.Nil` on errors.

 *  Use functions like `require.Foo` instead of `assert.Foo` when the test
    cannot continue if the condition is false.

 ###  <a id="recommended-reading" href="#recommended-reading">Recommended Reading</a>

 *  <https://github.com/golang/go/wiki/CodeReviewComments>.

 *  <https://github.com/golang/go/wiki/TestComments>.

 *  <https://go-proverbs.github.io/>

[constant errors]:          https://dave.cheney.net/2016/04/07/constant-errors
[Linus said]:               https://www.kernel.org/doc/html/v4.17/process/coding-style.html#indentation
[Text, Including Comments]: #text-including-comments



##  <a id="markdown" href="#markdown">Markdown</a>

 *  **TODO(a.garipov):** Define more Markdown conventions.

 *  Prefer triple-backtick preformatted code blocks to indented code blocks.

 *  Use asterisks and not underscores for bold and italic.

 *  Use either link references or link destinations only.  Put all link
    reference definitions at the end of the second-level block.



##  <a id="shell-scripting" href="#shell-scripting">Shell Scripting</a>

 *  Avoid bashisms and GNUisms, prefer POSIX features only.

 *  Prefer `'raw strings'` to `"double quoted strings"` whenever possible.

 *  Put spaces within `$( cmd )`, `$(( expr ))`, and `{ cmd; }`.

 *  Put utility flags in the ASCII order and **don't** group them together.  For
    example, `ls -1 -A -q`.

 *  `snake_case`, not `camelCase` for variables.  `kebab-case` for filenames.

 *  UPPERCASE names for external exported variables, lowercase for local,
    unexported ones.

 *  Use `set -e -f -u` and also `set -x` in verbose mode.

 *  Use `readonly` liberally.

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

##  <a id="text-including-comments" href="#text-including-comments">Text, Including Comments</a>

 *  End sentences with appropriate punctuation.

 *  Headers should be written with all initial letters capitalized, except for
    references to variable names that start with a lowercase letter.

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

##  <a id="yaml" href="#yaml">YAML</a>

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
