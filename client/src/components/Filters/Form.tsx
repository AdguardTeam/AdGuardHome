import React, { useEffect, useState, useRef } from 'react';
import { useForm, Controller, FormProvider, useWatch } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';
import { validatePath, validateRequiredValue } from '../../helpers/validators';

import { MODAL_OPEN_TIMEOUT, MODAL_TYPE } from '../../helpers/constants';
import filtersCatalog from '../../helpers/filters/filters';
import { FiltersList } from './FiltersList';
import { Input } from '../ui/Controls/Input';
import { fetchFilterTitle } from '../../actions/filtering';

type FormValues = {
    enabled: boolean;
    name: string;
    url: string;
};

const defaultValues: FormValues = {
    enabled: true,
    name: '',
    url: '',
};

type Props = {
    closeModal: () => void;
    onSubmit: (values: FormValues) => void;
    processingAddFilter: boolean;
    processingConfigFilter: boolean;
    whitelist?: boolean;
    modalType: string;
    toggleFilteringModal: ({ type }: { type?: keyof typeof MODAL_TYPE }) => void;
    selectedSources?: Record<string, boolean>;
    initialValues?: FormValues;
};

export const Form = ({
    closeModal,
    processingAddFilter,
    processingConfigFilter,
    whitelist,
    modalType,
    toggleFilteringModal,
    selectedSources,
    onSubmit,
    initialValues,
}: Props) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const [isFetchingTitle, setIsFetchingTitle] = useState(false);

    const methods = useForm({
        defaultValues: {
            ...defaultValues,
            ...initialValues,
        },
        mode: 'onBlur',
    });
    const { handleSubmit, control, setValue, getValues } = methods;

    // Watch URL field for changes
    const urlValue = useWatch({ control, name: 'url' });
    const nameValue = useWatch({ control, name: 'name' });
    const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);

    // Auto-fetch title from URL
    useEffect(() => {
        // Clear any existing timer
        if (debounceTimerRef.current) {
            clearTimeout(debounceTimerRef.current);
        }

        // Don't fetch if:
        // - URL is empty
        // - Name field already has a value (user has typed something)
        // - We're in edit mode (initialValues provided)
        // - URL doesn't pass basic validation
        if (!urlValue || nameValue || initialValues?.name) {
            return;
        }

        // Basic URL validation check
        const isValidUrl = validatePath(urlValue) === undefined;
        if (!isValidUrl) {
            return;
        }

        // Debounce: wait 800ms after user stops typing
        debounceTimerRef.current = setTimeout(async () => {
            setIsFetchingTitle(true);
            try {
                const title = await dispatch(fetchFilterTitle(urlValue) as any);
                // Only set the name if it's still empty (user hasn't typed anything)
                if (!getValues('name') && title) {
                    setValue('name', title, { shouldValidate: false });
                }
            } finally {
                setIsFetchingTitle(false);
            }
        }, 800);

        // Cleanup on unmount
        return () => {
            if (debounceTimerRef.current) {
                clearTimeout(debounceTimerRef.current);
            }
        };
    }, [urlValue, nameValue, dispatch, setValue, getValues, initialValues]);

    const openModal = (modalType: keyof typeof MODAL_TYPE, timeout = MODAL_OPEN_TIMEOUT) => {
        toggleFilteringModal(undefined);
        setTimeout(() => toggleFilteringModal({ type: modalType }), timeout);
    };

    const openFilteringListModal = () => openModal('CHOOSE_FILTERING_LIST');

    const openAddFiltersModal = () => openModal('ADD_FILTERS');

    return (
        <FormProvider {...methods}>
            <form onSubmit={handleSubmit(onSubmit)}>
                <div className="modal-body modal-body--filters">
                    {modalType === MODAL_TYPE.SELECT_MODAL_TYPE && (
                        <div className="d-flex justify-content-around">
                            <button
                                onClick={openFilteringListModal}
                                className="btn btn-success btn-standard mr-2 btn-large">
                                {t('choose_from_list')}
                            </button>

                            <button onClick={openAddFiltersModal} className="btn btn-primary btn-standard">
                                {t('add_custom_list')}
                            </button>
                        </div>
                    )}
                    {modalType === MODAL_TYPE.CHOOSE_FILTERING_LIST && (
                        <FiltersList
                            categories={filtersCatalog.categories}
                            filters={filtersCatalog.filters}
                            selectedSources={selectedSources}
                        />
                    )}
                    {modalType !== MODAL_TYPE.CHOOSE_FILTERING_LIST && modalType !== MODAL_TYPE.SELECT_MODAL_TYPE && (
                        <>
                            <div className="form__group">
                                <Controller
                                    name="name"
                                    control={control}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="filters_name"
                                            placeholder={
                                                isFetchingTitle
                                                    ? t('fetching_name_from_url')
                                                    : t('name_auto_fetch_hint')
                                            }
                                            error={fieldState.error?.message}
                                            trimOnBlur
                                        />
                                    )}
                                />
                            </div>

                            <div className="form__group">
                                <Controller
                                    name="url"
                                    control={control}
                                    rules={{ validate: { validateRequiredValue, validatePath } }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="filters_url"
                                            placeholder={t('enter_url_or_path_hint')}
                                            error={fieldState.error?.message}
                                            trimOnBlur
                                        />
                                    )}
                                />
                            </div>

                            <div className="form__description">
                                {whitelist ? t('enter_valid_allowlist') : t('enter_valid_blocklist')}
                            </div>
                        </>
                    )}
                </div>

                <div className="modal-footer">
                    <button type="button" className="btn btn-secondary" onClick={closeModal}>
                        {t('cancel_btn')}
                    </button>

                    {modalType !== MODAL_TYPE.SELECT_MODAL_TYPE && (
                        <button
                            type="submit"
                            data-testid="filters_save"
                            className="btn btn-success"
                            disabled={processingAddFilter || processingConfigFilter}>
                            {t('save_btn')}
                        </button>
                    )}
                </div>
            </form>
        </FormProvider>
    );
};
