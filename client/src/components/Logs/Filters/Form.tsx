import React, { useCallback, useEffect } from 'react';

import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';

import { useHistory } from 'react-router-dom';
import classNames from 'classnames';
import { useFormContext } from 'react-hook-form';
import queryString from 'query-string';

import {
    DEBOUNCE_FILTER_TIMEOUT,
    DEFAULT_LOGS_FILTER,
    RESPONSE_FILTER,
    RESPONSE_FILTER_QUERIES,
} from '../../../helpers/constants';
import { setLogsFilter } from '../../../actions/queryLogs';
import useDebounce from '../../../helpers/useDebounce';

import { getLogsUrlParams } from '../../../helpers/helpers';

import { SearchField } from './SearchField';
import { SearchFormValues } from '..';

type Props = {
    className?: string;
    setIsLoading: (value: boolean) => void;
};

export const Form = ({ className, setIsLoading }: Props) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const history = useHistory();

    const { register, watch, setValue } = useFormContext<SearchFormValues>();

    const excludeValue = watch('exclude');
    const responseStatusValue = watch('response_status');
    const isExclude = Boolean(excludeValue);

    const activeSearchField = isExclude ? 'exclude' : 'search';
    const searchValue = watch(activeSearchField);

    const [debouncedSearch, setDebouncedSearch] = useDebounce(searchValue.trim(), DEBOUNCE_FILTER_TIMEOUT);

    useEffect(() => {
        const searchParam = isExclude ? '' : debouncedSearch;
        const excludeParam = isExclude ? debouncedSearch : '';

        dispatch(
            setLogsFilter({
                response_status: responseStatusValue,
                search: searchParam,
                exclude: excludeParam,
            }),
        );

        history.replace(`${getLogsUrlParams(searchParam, responseStatusValue, excludeParam)}`);
    }, [responseStatusValue, debouncedSearch, isExclude]);

    useEffect(() => {
        if (responseStatusValue && !(responseStatusValue in RESPONSE_FILTER_QUERIES)) {
            setValue('response_status', DEFAULT_LOGS_FILTER.response_status);
        }
    }, [responseStatusValue, setValue]);

    useEffect(() => {
        const { search: searchUrlParam, exclude: excludeUrlParam } = queryString.parse(history.location.search);

        const searchParam = searchUrlParam ? searchUrlParam.toString() : '';
        const excludeParam = excludeUrlParam ? excludeUrlParam.toString() : '';

        if (excludeParam) {
            setValue('exclude', excludeParam);
            setValue('search', excludeParam);
        } else {
            setValue('search', searchParam);
            setValue('exclude', '');
        }
    }, [history.location.search, setValue]);

    const toggleExclude = useCallback(() => {
        const willExclude = !isExclude;
        const currentVal = searchValue;
        if (willExclude) {
            if (!currentVal) {
                return;
            }
            setValue('exclude', currentVal);
        } else {
            setValue('exclude', '');
        }
    }, [isExclude, searchValue, setValue]);

    const onInputClear = async () => {
        setIsLoading(true);
        setValue('search', '');
        setValue('exclude', '');
        history.push(getLogsUrlParams('', responseStatusValue, ''));
        setIsLoading(false);
    };

    const onEnterPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            setDebouncedSearch(searchValue);
        }
    };

    return (
        <form
            className="d-flex flex-wrap form-control--container"
            onSubmit={(e) => {
                e.preventDefault();
            }}>
            <div className="field__search">
                <button
                    type="button"
                    className={classNames(
                        'btn btn-icon logs__search-toggle',
                        { 'logs__search-toggle--active': isExclude },
                    )}
                    onClick={toggleExclude}
                    title={t(isExclude ? 'query_log_exclude_tooltip' : 'query_log_include_tooltip')}>
                    <svg className="icons icon--24">
                        <use xlinkHref="#magnifier" />
                    </svg>
                </button>
                <SearchField
                    data-testid={isExclude ? 'querylog_exclude' : 'querylog_search'}
                    value={searchValue}
                    handleChange={(val) => {
                        setValue(activeSearchField, val);
                        if (isExclude) {
                            setValue('search', val);
                        }
                    }}
                    onKeyDown={onEnterPress}
                    onClear={onInputClear}
                    placeholder={t(isExclude ? 'domain_to_exclude' : 'domain_or_client')}
                    tooltip={t(isExclude ? 'query_log_exclude_strict_search' : 'query_log_strict_search')}
                    className={classNames(
                        'form-control form-control--search form-control--transparent',
                        { 'form-control--exclude': isExclude },
                        className,
                    )}
                />
                {isExclude && (
                    <span className="logs__exclude-badge">
                        {t('exclude_mode')}
                    </span>
                )}
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
