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
        const { t } = this.props;
        return (
            <Card
                title={ t('custom_filter_rules') }
                subtitle={ t('custom_filter_rules_hint') }
            >
                <form onSubmit={this.handleSubmit}>
                    <textarea className="form-control form-control--textarea-large" value={this.props.userRules} onChange={this.handleChange} />
                    <div className="card-actions">
                        <button
                            className="btn btn-success btn-standard"
                            type="submit"
                            onClick={this.handleSubmit}
                        >
                            <Trans>apply_btn</Trans>
                        </button>
                    </div>
                </form>
                <hr/>
                <div className="list leading-loose">
                    <Trans>examples_title</Trans>:
                    <ol className="leading-loose">
                        <li>
                            <code>||example.org^</code> - { t('example_meaning_filter_block') }
                        </li>
                        <li>
                            <code> @@||example.org^</code> - { t('example_meaning_filter_whitelist') }
                        </li>
                        <li>
                            <code>127.0.0.1 example.org</code> - { t('example_meaning_host_block') }
                        </li>
                        <li>
                            <code>{ t('example_comment') }</code> - { t('example_comment_meaning') }
                        </li>
                        <li>
                            <code>{ t('example_comment_hash') }</code> - { t('example_comment_meaning') }
                        </li>
                        <li>
                            <code>/REGEX/</code> - { t('example_regex_meaning') }
                        </li>
                    </ol>
                </div>
            </Card>
        );
    }
}

UserRules.propTypes = {
    userRules: PropTypes.string,
    handleRulesChange: PropTypes.func,
    handleRulesSubmit: PropTypes.func,
    t: PropTypes.func,
};

export default withNamespaces()(UserRules);
