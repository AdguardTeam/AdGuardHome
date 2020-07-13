import React, { useEffect } from 'react';
import PropTypes from 'prop-types';
import { Field, reduxForm } from 'redux-form';
import { useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import { useHistory } from 'react-router-dom';
import classNames from 'classnames';
import {
    DEBOUNCE_FILTER_TIMEOUT,
    DEFAULT_LOGS_FILTER,
    FORM_NAME,
    RESPONSE_FILTER,
    RESPONSE_FILTER_QUERIES,
} from '../../../helpers/constants';
import IconTooltip from '../../ui/IconTooltip';
import { setLogsFilter } from '../../../actions/queryLogs';
import useDebounce from '../../../helpers/useDebounce';
import { createOnBlurHandler, getLogsUrlParams } from '../../../helpers/helpers';

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
    onClearInputClick,
    onKeyDown,
    normalizeOnBlur,
}) => {
    const onBlur = (event) => createOnBlurHandler(event, input, normalizeOnBlur);

    return <>
        <div className="input-group-search input-group-search__icon--magnifier">
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
            aria-label={placeholder}
            onKeyDown={onKeyDown}
            onBlur={onBlur}
        />
        <div
            className={classNames('input-group-search input-group-search__icon--cross', { invisible: input.value.length < 1 })}>
            <svg className="icons icon--smallest icon--gray" onClick={onClearInputClick}>
                <use xlinkHref="#cross" />
            </svg>
        </div>
        <span className="input-group-search input-group-search__icon--tooltip">
        <IconTooltip text={tooltip} type='tooltip-custom--logs' />
    </span>
        {!disabled
        && touched
        && (error && <span className="form__message form__message--error">{error}</span>)}
    </>;
};

renderFilterField.propTypes = {
    input: PropTypes.object.isRequired,
    id: PropTypes.string.isRequired,
    onClearInputClick: PropTypes.func.isRequired,
    className: PropTypes.string,
    placeholder: PropTypes.string,
    type: PropTypes.string,
    disabled: PropTypes.string,
    autoComplete: PropTypes.string,
    tooltip: PropTypes.string,
    onKeyDown: PropTypes.func,
    normalizeOnBlur: PropTypes.func,
    meta: PropTypes.shape({
        touched: PropTypes.bool,
        error: PropTypes.object,
    }).isRequired,
};

const FORM_NAMES = {
    search: 'search',
    response_status: 'response_status',
};

const Form = (props) => {
    const {
        className = '',
        responseStatusClass,
        setIsLoading,
        change,
    } = props;

    const { t } = useTranslation();
    const dispatch = useDispatch();
    const history = useHistory();

    const {
        response_status, search,
    } = useSelector((state) => state.form[FORM_NAME.LOGS_FILTER].values, shallowEqual);

    const [
        debouncedSearch,
        setDebouncedSearch,
    ] = useDebounce(search.trim(), DEBOUNCE_FILTER_TIMEOUT);

    useEffect(() => {
        dispatch(setLogsFilter({
            response_status,
            search: debouncedSearch,
        }));

        history.replace(`${getLogsUrlParams(debouncedSearch, response_status)}`);
    }, [response_status, debouncedSearch]);

    if (response_status && !(response_status in RESPONSE_FILTER_QUERIES)) {
        change(FORM_NAMES.response_status, DEFAULT_LOGS_FILTER[FORM_NAMES.response_status]);
    }

    const onInputClear = async () => {
        setIsLoading(true);
        setDebouncedSearch(DEFAULT_LOGS_FILTER[FORM_NAMES.search]);
        change(FORM_NAMES.search, DEFAULT_LOGS_FILTER[FORM_NAMES.search]);
        setIsLoading(false);
    };

    const onEnterPress = (e) => {
        if (e.key === 'Enter') {
            setDebouncedSearch(search);
        }
    };

    const normalizeOnBlur = (data) => data.trim();

    return (
        <form className="d-flex flex-wrap form-control--container"
              onSubmit={(e) => {
                  e.preventDefault();
              }}
        >
            <Field
                id={FORM_NAMES.search}
                name={FORM_NAMES.search}
                component={renderFilterField}
                type="text"
                className={classNames('form-control--search form-control--transparent', className)}
                placeholder={t('domain_or_client')}
                tooltip={t('query_log_strict_search')}
                onClearInputClick={onInputClear}
                onKeyDown={onEnterPress}
                normalizeOnBlur={normalizeOnBlur}
            />
            <div className="field__select">
                <Field
                    name={FORM_NAMES.response_status}
                    component="select"
                    className={classNames('form-control custom-select custom-select--logs custom-select__arrow--left ml-small form-control--transparent', responseStatusClass)}
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
    className: PropTypes.string,
    responseStatusClass: PropTypes.string,
    change: PropTypes.func.isRequired,
    setIsLoading: PropTypes.func.isRequired,
};

export default reduxForm({
    form: FORM_NAME.LOGS_FILTER,
    enableReinitialize: true,
})(Form);
