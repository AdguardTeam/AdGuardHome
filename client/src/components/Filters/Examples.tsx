import React, { Fragment } from 'react';
import { withTranslation, Trans } from 'react-i18next';

const Examples = () => (
    <Fragment>
        <div className="list leading-loose">
            <Trans>examples_title</Trans>:
            <ol className="leading-loose">
                <li>
                    <code>||example.org^</code>:<Trans>example_meaning_filter_block</Trans>
                </li>

                <li>
                    <code> @@||example.org^</code>:<Trans>example_meaning_filter_whitelist</Trans>
                </li>

                <li>
                    <code>127.0.0.1 example.org</code>:<Trans>example_meaning_host_block</Trans>
                </li>

                <li>
                    <code>
                        <Trans>example_comment</Trans>
                    </code>
                    :<Trans>example_comment_meaning</Trans>
                </li>

                <li>
                    <code>
                        <Trans>example_comment_hash</Trans>
                    </code>
                    :<Trans>example_comment_meaning</Trans>
                </li>

                <li>
                    <code>/REGEX/</code>:<Trans>example_regex_meaning</Trans>
                </li>
            </ol>
        </div>

        <p className="mt-1">
            <Trans
                components={[
                    <a
                        href="https://link.adtidy.org/forward.html?action=dns_kb_filtering_syntax&from=ui&app=home"
                        target="_blank"
                        rel="noopener noreferrer"
                        key="0">
                        link
                    </a>,
                ]}>
                filtering_rules_learn_more
            </Trans>
        </p>
    </Fragment>
);

export default withTranslation()(Examples);
