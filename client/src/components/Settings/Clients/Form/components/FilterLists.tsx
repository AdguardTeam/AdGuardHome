import React, { useEffect } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { Controller, useFormContext } from 'react-hook-form';
import { useDispatch, useSelector } from 'react-redux';
import { ClientForm } from '../types';
import { ServiceField } from '../../../../Filters/Services/ServiceField';
import { RootState } from '../../../../../initialState';
import { getFilteringStatus } from '../../../../../actions/filtering';
import { Filter } from '../../../../../helpers/helpers';

export const FilterLists = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const { watch, setValue, control } = useFormContext<ClientForm>();

    const useGlobalFilterLists = watch('use_global_filter_lists');

    const filters: Filter[] = useSelector((state: RootState) => state?.filtering?.filters) || [];
    const whitelistFilters: Filter[] =
        useSelector((state: RootState) => state?.filtering?.whitelistFilters) || [];

    useEffect(() => {
        dispatch(getFilteringStatus());
    }, [dispatch]);

    const handleToggleAllBlocklists = (isSelected: boolean) => {
        filters.forEach((filter) => setValue(`filter_list_ids.${filter.id}`, isSelected));
    };

    const handleToggleAllAllowlists = (isSelected: boolean) => {
        whitelistFilters.forEach((filter) => setValue(`allow_filter_list_ids.${filter.id}`, isSelected));
    };

    return (
        <div title={t('client_filter_lists')}>
            <div className="form__group">
                <Controller
                    name="use_global_filter_lists"
                    control={control}
                    render={({ field }) => (
                        <ServiceField
                            {...field}
                            data-testid="clients_use_global_filter_lists"
                            placeholder={t('use_global_filter_lists')}
                            className="service--global"
                        />
                    )}
                />

                <div className="form__desc mt-0 mb-2">
                    <Trans>client_filter_lists_desc</Trans>
                </div>

                {filters.length > 0 && (
                    <>
                        <div className="form__label mt-3">
                            <strong>
                                <Trans>dns_blocklists</Trans>
                            </strong>
                        </div>

                        <div className="row mb-2">
                            <div className="col-6">
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-block btn-sm"
                                    disabled={useGlobalFilterLists}
                                    onClick={() => handleToggleAllBlocklists(true)}>
                                    <Trans>select_all</Trans>
                                </button>
                            </div>

                            <div className="col-6">
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-block btn-sm"
                                    disabled={useGlobalFilterLists}
                                    onClick={() => handleToggleAllBlocklists(false)}>
                                    <Trans>deselect_all</Trans>
                                </button>
                            </div>
                        </div>

                        <div className="services services--half">
                            {filters.map((filter: Filter) => (
                                <Controller
                                    key={filter.id}
                                    name={`filter_list_ids.${filter.id}`}
                                    control={control}
                                    render={({ field }) => (
                                        <ServiceField
                                            {...field}
                                            data-testid={`clients_filter_${filter.id}`}
                                            placeholder={filter.name}
                                            disabled={useGlobalFilterLists}
                                        />
                                    )}
                                />
                            ))}
                        </div>
                    </>
                )}

                {whitelistFilters.length > 0 && (
                    <>
                        <div className="form__label mt-3">
                            <strong>
                                <Trans>dns_allowlists</Trans>
                            </strong>
                        </div>

                        <div className="row mb-2">
                            <div className="col-6">
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-block btn-sm"
                                    disabled={useGlobalFilterLists}
                                    onClick={() => handleToggleAllAllowlists(true)}>
                                    <Trans>select_all</Trans>
                                </button>
                            </div>

                            <div className="col-6">
                                <button
                                    type="button"
                                    className="btn btn-secondary btn-block btn-sm"
                                    disabled={useGlobalFilterLists}
                                    onClick={() => handleToggleAllAllowlists(false)}>
                                    <Trans>deselect_all</Trans>
                                </button>
                            </div>
                        </div>

                        <div className="services services--half">
                            {whitelistFilters.map((filter: Filter) => (
                                <Controller
                                    key={filter.id}
                                    name={`allow_filter_list_ids.${filter.id}`}
                                    control={control}
                                    render={({ field }) => (
                                        <ServiceField
                                            {...field}
                                            data-testid={`clients_allowfilter_${filter.id}`}
                                            placeholder={filter.name}
                                            disabled={useGlobalFilterLists}
                                        />
                                    )}
                                />
                            ))}
                        </div>
                    </>
                )}
            </div>
        </div>
    );
};
