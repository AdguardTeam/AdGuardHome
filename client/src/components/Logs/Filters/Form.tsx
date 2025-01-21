import React, { useEffect } from 'react';

import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';

import { useHistory } from 'react-router-dom';
import classNames from 'classnames';
import { useForm } from 'react-hook-form';
import {
    DEBOUNCE_FILTER_TIMEOUT,
    DEFAULT_LOGS_FILTER,
    RESPONSE_FILTER,
    RESPONSE_FILTER_QUERIES,
} from '../../../helpers/constants';
import { setLogsFilter } from '../../../actions/queryLogs';
import useDebounce from '../../../helpers/useDebounce';

import { createOnBlurHandler, getLogsUrlParams } from '../../../helpers/helpers';

import { SearchField } from './SearchField';

export type FormValues = {
    search: string;
    response_status: string;
};

type Props = {
    initialValues: FormValues;
    className?: string;
    setIsLoading: (value: boolean) => void;
};

export const Form = ({ initialValues, className, setIsLoading }: Props) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const history = useHistory();

    const { register, watch, setValue } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: {
            search: initialValues.search || DEFAULT_LOGS_FILTER.search,
            response_status: initialValues.response_status || DEFAULT_LOGS_FILTER.response_status,
        },
    });

    const searchValue = watch('search');
    const responseStatusValue = watch('response_status');

    const [debouncedSearch, setDebouncedSearch] = useDebounce(searchValue.trim(), DEBOUNCE_FILTER_TIMEOUT);

    useEffect(() => {
        dispatch(
            setLogsFilter({
                response_status: responseStatusValue,
                search: debouncedSearch,
            }),
        );

        history.replace(`${getLogsUrlParams(debouncedSearch, responseStatusValue)}`);
    }, [responseStatusValue, debouncedSearch]);

    useEffect(() => {
        if (responseStatusValue && !(responseStatusValue in RESPONSE_FILTER_QUERIES)) {
            setValue('response_status', DEFAULT_LOGS_FILTER.response_status);
        }
    }, [responseStatusValue, setValue]);

    const onInputClear = async () => {
        setIsLoading(true);
        setValue('search', DEFAULT_LOGS_FILTER.search);
        setIsLoading(false);
    };

    const onEnterPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            setDebouncedSearch(searchValue);
        }
    };

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>) =>
        createOnBlurHandler(
            e,
            {
                value: e.target.value,
                onChange: (v: string) => setValue('search', v),
            },
            (data: string) => data.trim(),
        );

    return (
        <form
            className="d-flex flex-wrap form-control--container"
            onSubmit={(e) => {
                e.preventDefault();
            }}>
            <div className="field__search">
                <SearchField
                    value={searchValue}
                    handleChange={(val) => setValue('search', val)}
                    onBlur={handleBlur}
                    onKeyDown={onEnterPress}
                    onClear={onInputClear}
                    placeholder={t('domain_or_client')}
                    tooltip={t('query_log_strict_search')}
                    className={classNames('form-control form-control--search form-control--transparent', className)}
                />
            </div>

            <div className="field__select">
                <select
                    {...register('response_status')}
                    className="form-control custom-select custom-select--logs custom-select__arrow--left form-control--transparent d-sm-block">
                    {Object.values(RESPONSE_FILTER).map(({ QUERY, LABEL, disabled }: any) => (
                        <option key={LABEL} value={QUERY} disabled={disabled}>
                            {t(LABEL)}
                        </option>
                    ))}
                </select>
            </div>
        </form>
    );
};
