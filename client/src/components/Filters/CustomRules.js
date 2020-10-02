import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import classnames from 'classnames';
import Card from '../ui/Card';
import PageTitle from '../ui/PageTitle';
import Examples from './Examples';
import Check from './Check';
import { getTextareaCommentsHighlight, syncScroll } from '../../helpers/highlightTextareaComments';
import { COMMENT_LINE_DEFAULT_TOKEN, isFirefox } from '../../helpers/constants';
import '../ui/texareaCommentsHighlight.css';

class CustomRules extends Component {
    ref = React.createRef();

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

    onScroll = (e) => syncScroll(e, this.ref)

    render() {
        const {
            t,
            filtering: {
                userRules,
            },
        } = this.props;

        return (
            <>
                <PageTitle title={t('custom_filtering_rules')} />
                <Card
                    subtitle={t('custom_filter_rules_hint')}
                >
                    <form onSubmit={this.handleSubmit}>
                        <div className={classnames('col-12 text-edit-container form-control--textarea-large', {
                            'mb-4': !isFirefox,
                            'mb-6': isFirefox,
                        })}>
                        <textarea
                                className={classnames('form-control font-monospace text-input form-control--textarea-large', {
                                    'text-input--largest': isFirefox,
                                })}
                                value={userRules}
                                onChange={this.handleChange}
                                onScroll={this.onScroll}
                        />
                            {getTextareaCommentsHighlight(
                                this.ref,
                                userRules,
                                classnames({ 'form-control--textarea-large': isFirefox }),
                                [COMMENT_LINE_DEFAULT_TOKEN, '!'],
                            )}
                        </div>
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
                <Check onSubmit={this.handleCheck} />
            </>
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

export default withTranslation()(CustomRules);
