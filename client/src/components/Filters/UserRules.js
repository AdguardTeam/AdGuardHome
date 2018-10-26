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
                title={ t('Custom filtering rules') }
                subtitle={ t('Enter one rule on a line. You can use either adblock rules or hosts files syntax.') }
            >
                <form onSubmit={this.handleSubmit}>
                    <textarea className="form-control form-control--textarea-large" value={this.props.userRules} onChange={this.handleChange} />
                    <div className="card-actions">
                        <button
                            className="btn btn-success btn-standart"
                            type="submit"
                            onClick={this.handleSubmit}
                        >
                            <Trans>Apply</Trans>
                        </button>
                    </div>
                </form>
                <hr/>
                <div className="list leading-loose">
                    <Trans>Examples</Trans>:
                    <ol className="leading-loose">
                        <li>
                            <code>||example.org^</code> - { t('block access to the example.org domain and all its subdomains') }
                        </li>
                        <li>
                            <code> @@||example.org^</code> - { t('unblock access to the example.org domain and all its subdomains') }
                        </li>
                        <li>
                            <code>127.0.0.1 example.org</code> - { t('AdGuard Home will now return 127.0.0.1 address for the example.org domain (but not its subdomains).') }
                        </li>
                        <li>
                            <code>{ t('! Here goes a comment') }</code> - { t('just a comment') }
                        </li>
                        <li>
                            <code>{ t('# Also a comment') }</code> - { t('just a comment') }
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
