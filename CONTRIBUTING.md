# Contributing to AdGuard Home

If you want to contribute to AdGuard Home by filing or commenting on an issue or opening a pull request, please follow the instructions below.

## General recommendations

Please don’t:

- post comments like “+1” or “this”.  Use the :+1: reaction on the issue instead, as this allows us to actually see the level of support for issues.

- file issues about localization errors or send localization updates as PRs.  We’re using [CrowdIn] to manage our translations and we generally update them before each Beta and Release build.  You can learn more about translating AdGuard products [in our Knowledge Base][kb-trans].

- file issues about a particular filtering-rule list misbehaving.  These are tracked through the [separate form for filtering issues][form].

- send or request updates to filtering-rule lists, such as the ones for the Blocked Services feature or the list of approved filtering-rule lists.  We update them from the [separate repository][hostlist] once before each Beta and Release build.

Please do:

- follow the template instructions and provide data for reproducing issues.

- write the title of your issue or pull request in English.  Any language is fine in the body, but it is important to keep the title in English to make it easier for people and bots to look up duplicated issues.

[CrowdIn]:  https://crowdin.com/project/adguard-applications/en#/adguard-home
[form]:     https://link.adtidy.org/forward.html?action=report&app=home&from=github
[hostlist]: https://github.com/AdguardTeam/HostlistsRegistry
[kb-trans]: https://kb.adguard.com/en/general/adguard-translations

## Issues

### Search first

Please make sure that the issue is not a duplicate or a question.  If it’s a duplicate, please react to the original issue with a thumbs up.  If it’s a question, please look through our [Wiki] and, if you haven’t found the answer, post it to the GitHub [Discussions] page.

[Discussions]: https://github.com/AdguardTeam/AdGuardHome/discussions/categories/q-a
[Wiki]:        https://github.com/AdguardTeam/AdGuardHome/wiki

### Follow the issue template

Developers need to be able to reproduce the faulty behavior in order to fix an issue, so please make sure that you follow the instructions in the issue template carefully.

## Pull requests

### Discuss your changes first

Please discuss your changes by opening an issue.  The maintainers should evaluate your proposal, and it’s generally better if that’s done before any code is written.

### Review your changes for style

We have a set of [code guidelines][hacking] that we expect the code to follow.  Please make sure you follow it.

[hacking]: https://github.com/AdguardTeam/CodeGuidelines/blob/master/Go/Go.md

### Test your changes

Make sure that it passes linters and tests by running the corresponding Make targets.  For backend changes, it’s `make go-check`.  For frontend, run `make js-lint`.

Additionally, a manual test is often required.  While we’re constantly working on improving our test suites, they’re still not as good as we’d like them to be.
