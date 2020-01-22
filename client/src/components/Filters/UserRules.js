import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import Card from '../ui/Card';

class UserRules extends Component {
    handleChange = (e) => {
        const { value } = e.currentTarget;
        this.props.handleRulesChange(value);
    };

    handleSubmit = (e) => {
        e.preventDefault();
        this.props.handleRulesSubmit();
    };

    render() {
        const { t, userRules } = this.props;
        return (
            <Card title={t('custom_filter_rules')} subtitle={t('custom_filter_rules_hint')}>
                <form onSubmit={this.handleSubmit}>
                    <textarea
                        className="form-control form-control--textarea-large"
                        value={userRules}
                        onChange={this.handleChange}
                    />
                    <div className="card-actions">
                        <button
                            className="btn btn-success btn-standard btn-large"
                            type="submit"
                            onClick={this.handleSubmit}
                        >
                            <Trans>apply_btn</Trans>
                        </button>
                    </div>
                </form>
                <hr />
                <div className="list leading-loose">
                    <Trans>examples_title</Trans>:
                    <ol className="leading-loose">
                        <li>
                            <code>||example.org^</code> –&nbsp;
                            <Trans>example_meaning_filter_block</Trans>
                        </li>
                        <li>
                            <code> @@||example.org^</code> –&nbsp;
                            <Trans>example_meaning_filter_whitelist</Trans>
                        </li>
                        <li>
                            <code>127.0.0.1 example.org</code> –&nbsp;
                            <Trans>example_meaning_host_block</Trans>
                        </li>
                        <li>
                            <code><Trans>example_comment</Trans></code> –&nbsp;
                            <Trans>example_comment_meaning</Trans>
                        </li>
                        <li>
                            <code><Trans>example_comment_hash</Trans></code> –&nbsp;
                            <Trans>example_comment_meaning</Trans>
                        </li>
                        <li>
                            <code>/REGEX/</code> –&nbsp;
                            <Trans>example_regex_meaning</Trans>
                        </li>
                    </ol>
                </div>
                <p className="mt-1">
                    <Trans
                        components={[
                            <a
                                href="https://github.com/AdguardTeam/AdGuardHome/wiki/Hosts-Blocklists"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="0"
                            >
                                link
                            </a>,
                        ]}
                    >
                        filtering_rules_learn_more
                    </Trans>
                </p>
            </Card>
        );
    }
}

UserRules.propTypes = {
    userRules: PropTypes.string.isRequired,
    handleRulesChange: PropTypes.func.isRequired,
    handleRulesSubmit: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(UserRules);
