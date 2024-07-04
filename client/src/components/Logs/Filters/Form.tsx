import React, { useEffect } from 'react';

import { Field, type InjectedFormProps, reduxForm } from 'redux-form';
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
import { setLogsFilter } from '../../../actions/queryLogs';
import useDebounce from '../../../helpers/useDebounce';

import { createOnBlurHandler, getLogsUrlParams } from '../../../helpers/helpers';

import Tooltip from '../../ui/Tooltip';
import { RootState } from '../../../initialState';

interface renderFilterFieldProps {
    input: {
        value: string;
    };
    id: string;
    onClearInputClick: (...args: unknown[]) => unknown;
    className?: string;
    placeholder?: string;
    type?: string;
    disabled?: boolean;
    autoComplete?: string;
    tooltip?: string;
    onKeyDown?: (...args: unknown[]) => unknown;
    normalizeOnBlur?: (...args: unknown[]) => unknown;
    meta: {
        touched?: boolean;
        error?: object;
    };
}

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
}: renderFilterFieldProps) => {
    const onBlur = (event: any) => createOnBlurHandler(event, input, normalizeOnBlur);

    return (
        <>
            <div className="input-group-search input-group-search__icon--magnifier">
                <svg className="icons icon--24 icon--gray">
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
                className={classNames('input-group-search input-group-search__icon--cross', {
                    invisible: input.value.length < 1,
                })}>
                <svg className="icons icon--20 icon--gray" onClick={onClearInputClick}>
                    <use xlinkHref="#cross" />
                </svg>
            </div>

            <span className="input-group-search input-group-search__icon--tooltip">
                <Tooltip content={tooltip} className="tooltip-container">
                    <svg className="icons icon--20 icon--gray">
                        <use xlinkHref="#question" />
                    </svg>
                </Tooltip>
            </span>
            {!disabled && touched && error && <span className="form__message form__message--error">{error}</span>}
        </>
    );
};

const FORM_NAMES = {
    search: 'search',
    response_status: 'response_status',
};

type FiltersFormProps = {
    className?: string;
    responseStatusClass?: string;
    setIsLoading: (...args: unknown[]) => unknown;
};

const Form = (props: FiltersFormProps & InjectedFormProps) => {
    const { className = '', responseStatusClass, setIsLoading, change } = props;

    const { t } = useTranslation();
    const dispatch = useDispatch();
    const history = useHistory();

    const { response_status, search } = useSelector(
        (state: RootState) => state?.form[FORM_NAME.LOGS_FILTER].values,
        shallowEqual,
    );

    const [debouncedSearch, setDebouncedSearch] = useDebounce(search.trim(), DEBOUNCE_FILTER_TIMEOUT);

    useEffect(() => {
        dispatch(
            setLogsFilter({
                response_status,
                search: debouncedSearch,
            }),
        );

        history.replace(`${getLogsUrlParams(debouncedSearch, response_status)}`);
    }, [response_status, debouncedSearch]);

    if (response_status && !(response_status in RESPONSE_FILTER_QUERIES)) {
        change(FORM_NAMES.response_status, DEFAULT_LOGS_FILTER[FORM_NAMES.response_status]);
    }

    const onInputClear = async () => {
        setIsLoading(true);
        change(FORM_NAMES.search, DEFAULT_LOGS_FILTER[FORM_NAMES.search]);
        setIsLoading(false);
    };

    const onEnterPress = (e: any) => {
        if (e.key === 'Enter') {
            setDebouncedSearch(search);
        }
    };

    const normalizeOnBlur = (data: any) => data.trim();

    return (
        <form
            className="d-flex flex-wrap form-control--container"
            onSubmit={(e) => {
                e.preventDefault();
            }}>
            <div className="field__search">
                <Field
                    id={FORM_NAMES.search}
                    name={FORM_NAMES.search}
                    component={renderFilterField}
                    type="text"
                    className={classNames('form-control form-control--search form-control--transparent', className)}
                    placeholder={t('domain_or_client')}
                    tooltip={t('query_log_strict_search')}
                    onClearInputClick={onInputClear}
                    onKeyDown={onEnterPress}
                    normalizeOnBlur={normalizeOnBlur}
                />
            </div>

            <div className="field__select">
                <Field
                    name={FORM_NAMES.response_status}
                    component="select"
                    className={classNames(
                        'form-control custom-select custom-select--logs custom-select__arrow--left form-control--transparent',
                        responseStatusClass,
                    )}>
                    {Object.values(RESPONSE_FILTER).map(({ QUERY, LABEL, disabled }: any) => (
                        <option key={LABEL} value={QUERY} disabled={disabled}>
                            {t(LABEL)}
                        </option>
                    ))}
                </Field>
            </div>
        </form>
    );
};

export const FiltersForm = reduxForm<Record<string, any>, FiltersFormProps>({
    form: FORM_NAME.LOGS_FILTER,
    enableReinitialize: true,
})(Form);
