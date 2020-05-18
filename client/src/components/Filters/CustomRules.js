import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Card from '../ui/Card';
import PageTitle from '../ui/PageTitle';
import Examples from './Examples';
import Check from './Check';

class CustomRules extends Component {
    componentDidMount() {
        this.props.getFilteringStatus();
    }

    handleChange = (e) => {
        const { value } = e.currentTarget;
        this.handleRulesChange(value);
    };

    handleSubmit = (e) => {
        e.preventDefault();
        this.handleRulesSubmit();
    };

    handleRulesChange = (value) => {
        this.props.handleRulesChange({ userRules: value });
    };

    handleRulesSubmit = () => {
        this.props.setRules(this.props.filtering.userRules);
    };

    handleCheck = (values) => {
        this.props.checkHost(values);
    };

    render() {
        const {
            t,
            filtering: {
                filters,
                whitelistFilters,
                userRules,
                processingCheck,
                check,
            },
        } = this.props;

        return (
            <Fragment>
                <PageTitle title={t('custom_filtering_rules')} />
                <Card
                    subtitle={t('custom_filter_rules_hint')}
                >
                    <form onSubmit={this.handleSubmit}>
                        <textarea
                            className="form-control form-control--textarea-large font-monospace"
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
                    <Examples />
                </Card>
                <Check
                    filters={filters}
                    whitelistFilters={whitelistFilters}
                    check={check}
                    onSubmit={this.handleCheck}
                    processing={processingCheck}
                />
            </Fragment>
        );
    }
}

CustomRules.propTypes = {
    filtering: PropTypes.object.isRequired,
    setRules: PropTypes.func.isRequired,
    checkHost: PropTypes.func.isRequired,
    getFilteringStatus: PropTypes.func.isRequired,
    handleRulesChange: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(CustomRules);
