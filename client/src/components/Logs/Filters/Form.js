import React from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { useTranslation } from 'react-i18next';
import debounce from 'lodash/debounce';
import { DEBOUNCE_FILTER_TIMEOUT, FORM_NAME, RESPONSE_FILTER } from '../../../helpers/constants';
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
}) => <>
    <div className="input-group-search">
        <svg className="icons icon--small icon--gray">
            <use xlinkHref="#magnifier" />
        </svg>
    </div>
    <input
        {...input}
        id={id}
        placeholder={placeholder}
        type={type}
        className={className}
        disabled={disabled}
        autoComplete={autoComplete}
        aria-label={placeholder} />
    <span className="logs__notice">
                <Tooltip text={tooltip} type='tooltip-custom--logs' />
            </span>
    {!disabled
    && touched
    && (error && <span className="form__message form__message--error">{error}</span>)}
</>;

renderFilterField.propTypes = {
    input: PropTypes.object.isRequired,
    id: PropTypes.string.isRequired,
    className: PropTypes.string,
    placeholder: PropTypes.string,
    type: PropTypes.string,
    disabled: PropTypes.string,
    autoComplete: PropTypes.string,
    tooltip: PropTypes.string,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.object,
    }).isRequired,
};

const Form = (props) => {
    const {
        className = '',
        responseStatusClass,
        submit,
    } = props;

    const [t] = useTranslation();

    const debouncedSubmit = debounce(submit, DEBOUNCE_FILTER_TIMEOUT);
    const zeroDelaySubmit = () => setTimeout(submit, 0);

    return (
        <form className="d-flex flex-wrap form-control--container"
              onSubmit={(e) => {
                  e.preventDefault();
                  zeroDelaySubmit();
                  debouncedSubmit.cancel();
              }}
        >
            <Field
                id="search"
                name="search"
                component={renderFilterField}
                type="text"
                className={`form-control--search form-control--transparent ${className}`}
                placeholder={t('domain_or_client')}
                tooltip={t('query_log_strict_search')}
                onChange={debouncedSubmit}
            />
            <div className="field__select">
                <Field
                    name="response_status"
                    component="select"
                    className={`form-control custom-select custom-select--logs custom-select__arrow--left ml-small form-control--transparent ${responseStatusClass}`}
                    onChange={zeroDelaySubmit}
                >
                    {Object.values(RESPONSE_FILTER)
                        .map(({
                            query, label, disabled,
                        }) => <option key={label} value={query}
                                      disabled={disabled}>{t(label)}</option>)}
                </Field>
            </div>
        </form>
    );
};

Form.propTypes = {
    handleChange: PropTypes.func,
    className: PropTypes.string,
    responseStatusClass: PropTypes.string,
    submit: PropTypes.func.isRequired,
};

export default reduxForm({
    form: FORM_NAME.LOGS_FILTER,
})(Form);
