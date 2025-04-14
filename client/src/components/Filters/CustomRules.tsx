import React, { Component } from 'react';
import { Trans, withTranslation } from 'react-i18next';

import Card from '../ui/Card';

import PageTitle from '../ui/PageTitle';

import Examples from './Examples';

import Check, { FilteringCheckFormValues } from './Check';

import { getTextareaCommentsHighlight, syncScroll } from '../../helpers/highlightTextareaComments';
import { COMMENT_LINE_DEFAULT_TOKEN } from '../../helpers/constants';
import '../ui/texareaCommentsHighlight.css';
import { FilteringData } from '../../initialState';

interface CustomRulesProps {
    filtering: FilteringData;
    setRules: (...args: unknown[]) => unknown;
    checkHost: (...args: unknown[]) => string;
    getFilteringStatus: (...args: unknown[]) => unknown;
    handleRulesChange: (...args: unknown[]) => unknown;
    t: (...args: unknown[]) => string;
}

class CustomRules extends Component<CustomRulesProps> {
    ref = React.createRef();

    componentDidMount() {
        this.props.getFilteringStatus();
    }

    handleChange = (e: any) => {
        const { value } = e.currentTarget;
        this.handleRulesChange(value);
    };

    handleSubmit = (e: any) => {
        e.preventDefault();
        this.handleRulesSubmit();
    };

    handleRulesChange = (value: any) => {
        this.props.handleRulesChange({ userRules: value });
    };

    handleRulesSubmit = () => {
        this.props.setRules(this.props.filtering.userRules);
    };

    handleCheck = (values: FilteringCheckFormValues) => {
        const params: FilteringCheckFormValues = { name: values.name };

        if (values.client) {
            params.client = values.client;
        }

        if (values.qtype) {
            params.qtype = values.qtype;
        }

        this.props.checkHost(params);
    };

    onScroll = (e: any) => syncScroll(e, this.ref);

    render() {
        const {
            t,
            filtering: { userRules },
        } = this.props;

        return (
            <>
                <PageTitle title={t('custom_filtering_rules')} />

                <Card subtitle={t('custom_filter_rules_hint')}>
                    <form onSubmit={this.handleSubmit}>
                        <div className="text-edit-container mb-4">
                            <textarea
                                data-testid="custom_rule_textarea"
                                className="form-control font-monospace text-input"
                                value={userRules}
                                onChange={this.handleChange}
                                onScroll={this.onScroll}
                            />
                            {getTextareaCommentsHighlight(this.ref, userRules, [
                                COMMENT_LINE_DEFAULT_TOKEN,
                                '!',
                            ])}
                        </div>

                        <div className="card-actions">
                            <button
                                data-testid="apply_custom_rule"
                                className="btn btn-success btn-standard btn-large"
                                type="submit"
                                onClick={this.handleSubmit}>
                                <Trans>apply_btn</Trans>
                            </button>
                        </div>
                    </form>

                    <hr />

                    <Examples />
                </Card>

                <Check onSubmit={this.handleCheck} />
            </>
        );
    }
}

export default withTranslation()(CustomRules);
