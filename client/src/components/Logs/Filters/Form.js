import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { withNamespaces, Trans } from 'react-i18next';
import flow from 'lodash/flow';

import { renderField } from '../../../helpers/form';
import { RESPONSE_FILTER } from '../../../helpers/constants';
import Tooltip from '../../ui/Tooltip';

const renderFilterField = ({
    input,
    id,
    className,
    placeholder,
    type,
    disabled,
    autoComplete,
    tooltip,
    meta: { touched, error },
}) => (
    <Fragment>
        <div className="logs__input-wrap">
            <input
                {...input}
                id={id}
                placeholder={placeholder}
                type={type}
                className={className}
                disabled={disabled}
                autoComplete={autoComplete}
            />
            <span className="logs__notice">
                <Tooltip text={tooltip} type='tooltip-custom--logs' />
            </span>
            {!disabled &&
                touched &&
                (error && <span className="form__message form__message--error">{error}</span>)}
        </div>
    </Fragment>
);

const Form = (props) => {
    const {
        t,
        handleChange,
    } = props;

    return (
        <form onSubmit={handleChange}>
            <div className="row">
                <div className="col-6 col-sm-3 my-2">
                    <Field
                        id="filter_domain"
                        name="filter_domain"
                        component={renderFilterField}
                        type="text"
                        className="form-control"
                        placeholder={t('domain_name_table_header')}
                        tooltip={t('query_log_strict_search')}
                        onChange={handleChange}
                    />
                </div>
                <div className="col-6 col-sm-3 my-2">
                    <Field
                        id="filter_question_type"
                        name="filter_question_type"
                        component={renderField}
                        type="text"
                        className="form-control"
                        placeholder={t('type_table_header')}
                        onChange={handleChange}
                    />
                </div>
                <div className="col-6 col-sm-3 my-2">
                    <Field
                        name="filter_response_status"
                        component="select"
                        className="form-control custom-select"
                    >
                        <option value={RESPONSE_FILTER.ALL}>
                            <Trans>show_all_filter_type</Trans>
                        </option>
                        <option value={RESPONSE_FILTER.FILTERED}>
                            <Trans>show_filtered_type</Trans>
                        </option>
                    </Field>
                </div>
                <div className="col-6 col-sm-3 my-2">
                    <Field
                        id="filter_client"
                        name="filter_client"
                        component={renderFilterField}
                        type="text"
                        className="form-control"
                        placeholder={t('client_table_header')}
                        tooltip={t('query_log_strict_search')}
                        onChange={handleChange}
                    />
                </div>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleChange: PropTypes.func,
    t: PropTypes.func.isRequired,
};

export default flow([
    withNamespaces(),
    reduxForm({
        form: 'logsFilterForm',
    }),
])(Form);
