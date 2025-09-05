import React, { useState, useEffect } from 'react';
import { useForm, FormProvider } from 'react-hook-form';

import { MODAL_TYPE, TAB_TYPE } from 'panel/helpers/constants';
import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { FormContent } from './blocks/FormContent';

export type FormValues = {
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
    modalType: string;
    toggleFilteringModal: ({ type }: { type?: keyof typeof MODAL_TYPE }) => void;
    selectedSources?: Record<string, boolean>;
    initialValues?: Partial<FormValues>;
};

export const Form = ({
    closeModal,
    processingAddFilter,
    processingConfigFilter,
    modalType,
    selectedSources,
    onSubmit,
    initialValues,
}: Props) => {
    const methods = useForm({
        defaultValues: {
            ...defaultValues,
            ...initialValues,
        },
        mode: 'onBlur',
    });
    const { handleSubmit, reset } = methods;

    useEffect(() => {
        reset({
            ...defaultValues,
            ...initialValues,
        });
    }, [initialValues, reset]);

    const [activeTab, setActiveTab] = useState<string>(TAB_TYPE.LIST);

    const handleCancel = () => {
        closeModal();
    };

    const handleFormSubmit = (values: FormValues) => {
        onSubmit(values);
    };

    useEffect(() => {
        return () => {
            reset(defaultValues);
        };
    }, [reset]);

    return (
        <FormProvider {...methods}>
            <form onSubmit={handleSubmit(handleFormSubmit)}>
                <div className={theme.dialog.description}>{intl.getMessage('blocklist_add_desc')}</div>

                <div>
                    <FormContent
                        modalType={modalType}
                        selectedSources={selectedSources}
                        activeTab={activeTab}
                        onTabChange={setActiveTab}
                    />
                </div>

                <div className={theme.dialog.footer}>
                    <Button
                        type="submit"
                        id="filters_save"
                        variant="primary"
                        size="small"
                        disabled={processingAddFilter || processingConfigFilter}
                        className={theme.dialog.button}>
                        {intl.getMessage('save')}
                    </Button>

                    <Button
                        type="button"
                        id="filters_cancel"
                        variant="secondary"
                        size="small"
                        onClick={handleCancel}
                        className={theme.dialog.button}>
                        {intl.getMessage('cancel')}
                    </Button>
                </div>
            </form>
        </FormProvider>
    );
};
