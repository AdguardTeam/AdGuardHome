import React from 'react';
import { useForm, Controller, FormProvider } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { validatePath, validateRequiredValue } from '../../helpers/validators';

import { MODAL_OPEN_TIMEOUT, MODAL_TYPE } from '../../helpers/constants';
import filtersCatalog from '../../helpers/filters/filters';
import { FiltersList } from './FiltersList';

type FormValues = {
    enabled: boolean;
    name: string;
    url: string;
};

type Props = {
    closeModal: (...args: unknown[]) => void;
    onSubmit: (...args: unknown[]) => void;
    processingAddFilter: boolean;
    processingConfigFilter: boolean;
    whitelist?: boolean;
    modalType: string;
    toggleFilteringModal: (...args: unknown[]) => void;
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

    const methods = useForm({ defaultValues: initialValues });
    const { handleSubmit, control } = methods;

    const openModal = (modalType: any, timeout = MODAL_OPEN_TIMEOUT) => {
        toggleFilteringModal();
        setTimeout(() => toggleFilteringModal({ type: modalType }), timeout);
    };

    const openFilteringListModal = () => openModal(MODAL_TYPE.CHOOSE_FILTERING_LIST);

    const openAddFiltersModal = () => openModal(MODAL_TYPE.ADD_FILTERS);

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
                                    render={({ field }) => (
                                        <input
                                            {...field}
                                            type="text"
                                            className="form-control"
                                            placeholder={t('enter_name_hint')}
                                            onBlur={(e) => field.onChange(e.target.value.trim())}
                                        />
                                    )}
                                />
                            </div>

                            <div className="form__group">
                                <Controller
                                    name="url"
                                    control={control}
                                    rules={{ validate: { validateRequiredValue, validatePath } }}
                                    render={({ field }) => (
                                        <input
                                            {...field}
                                            type="text"
                                            className="form-control"
                                            placeholder={t('enter_url_or_path_hint')}
                                            onBlur={(e) => field.onChange(e.target.value.trim())}
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
