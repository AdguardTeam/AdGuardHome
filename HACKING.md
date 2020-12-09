 #  *AdGuardHome* Developer Guidelines

As of **December 2020**, this document is partially a work-in-progress, but
should still be followed.  Some of the rules aren't enforced as thoroughly or
remain broken in old code, but this is still the place to find out about what we
**want** our code to look like.

The rules are mostly sorted in the alphabetical order.

##  *Git*

 *  Call your branches either `NNNN-fix-foo` (where `NNNN` is the ID of the
    *GitHub* issue you worked on in this branch) or just `fix-foo` if there was
    no *GitHub* issue.

 *  Follow the commit message header format:

    ```none
    pkg: fix the network error logging issue
    ```

    Where `pkg` is the package where most changes took place.  If there are
    several such packages, or the change is top-level only, write `all`.

 *  Keep your commit messages, including headers, to eighty (**80**) columns.

 *  Only use lowercase letters in your commit message headers.  The rest of the
    message should follow the plain text conventions below.

    The only exceptions are direct mentions of identifiers from the source code
    and filenames like `HACKING.md`.

##  *Go*

> Not Golang, not GO, not GOLANG, not GoLang. It is Go in natural language,
> golang for others.

— [@rakyll](https://twitter.com/rakyll/status/1229850223184269312)

 ###  Code And Naming

 *  Avoid `goto`.

 *  Avoid `init` and use explicit initialization functions instead.

 *  Avoid `new`, especially with structs.

 *  Constructors should validate their arguments and return meaningful errors.
    As a corollary, avoid lazy initialization.

 *  Don't use naked `return`s.

 *  Don't use underscores in file and package names, unless they're build tags
    or for tests.  This is to prevent accidental build errors with weird tags.

 *  Don't write code with more than four (**4**) levels of indentation.  Just
    like [Linus said], plus an additional level for an occasional error check or
    struct initialization.

 *  Eschew external dependencies, including transitive, unless
    absolutely necessary.

 *  Name benchmarks and tests using the same convention as examples.  For
    example:

    ```go
    func TestFunction(t *testing.T) { /* … */ }
    func TestFunction_suffix(t *testing.T) { /* … */ }
    func TestType_Method(t *testing.T) { /* … */ }
    func TestType_Method_suffix(t *testing.T) { /* … */ }
    ```

 *  Name the deferred errors (e.g. when closing something) `cerr`.

 *  No shadowing, since it can often lead to subtle bugs, especially with
    errors.

 *  Prefer constants to variables where possible.  Reduce global variables.  Use
    [constant errors] instead of `errors.New`.

 *  Use linters.

 *  Use named returns to improve readability of function signatures.

 *  Write logs and error messages in lowercase only to make it easier to `grep`
    logs and error messages without using the `-i` flag.

[constant errors]: https://dave.cheney.net/2016/04/07/constant-errors
[Linus said]:      https://www.kernel.org/doc/html/v4.17/process/coding-style.html#indentation

 ###  Commenting

 *  See also the *Text, Including Comments* section below.

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

 ###  Formatting

 *  Add an empty line before `break`, `continue`, `fallthrough`, and `return`,
    unless it's the only statement in that block.

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

 ###  Recommended Reading

 *  <https://github.com/golang/go/wiki/CodeReviewComments>.

 *  <https://github.com/golang/go/wiki/TestComments>.

 *  <https://go-proverbs.github.io/>

##  *Markdown*

 *  **TODO(a.garipov):** Define our *Markdown* conventions.

##  Shell Scripting

 *  Avoid bashisms, prefer *POSIX* features only.

 *  Prefer `'raw strings'` to `"double quoted strings"` whenever possible.

 *  Put spaces within `$( cmd )`, `$(( expr ))`, and `{ cmd; }`.

 *  Use `set -e -f -u` and also `set -x` in verbose mode.

 *  Use the `"$var"` form instead of the `$var` form, unless word splitting is
    required.

##  Text, Including Comments

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

 *  Use the serial comma (a.k.a. *Oxford* comma) to improve comprehension,
    decrease ambiguity, and use a common standard.

 *  Write todos like this:

    ```go
    // TODO(usr1): Fix the frobulation issue.
    ```

    Or, if several people need to look at the code:

    ```go
    // TODO(usr1, usr2): Fix the frobulation issue.
    ```

##  *YAML*

 *  **TODO(a.garipov):** Define naming conventions for schema names in our
    *OpenAPI* *YAML* file.  And just generally OpenAPI conventions.

 *  **TODO(a.garipov):** Find a *YAML* formatter or write our own.

 *  All strings, including keys, must be quoted.  Reason: the [*NO-rway Law*].

 *  Indent with two (**2**) spaces.  *YAML* documents can get pretty
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
    *GitHub Actions*).

 *  Use `>` for multiline strings, unless you need to keep the line breaks.

[*NO-rway Law*]: https://news.ycombinator.com/item?id=17359376
