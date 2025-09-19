import React, { useEffect } from 'react';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { MODAL_TYPE } from 'panel/helpers/constants';

import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { useDispatch, useSelector } from 'react-redux';
import { closeModal } from 'panel/reducers/modals';
import theme from 'panel/lib/theme';
import { Button } from 'panel/common/ui/Button';
import { Controller, FormProvider, useForm } from 'react-hook-form';
import { RootState } from 'panel/initialState';
import { addFilter, editFilter } from 'panel/actions/filtering';
import { Input } from 'panel/common/controls/Input';
import { validatePath, validateRequiredValue } from 'panel/helpers/validators';

type FormValues = {
    name: string;
    url: string;
    enabled?: boolean;
};

const defaultValues: FormValues = {
    name: '',
    url: '',
};

type ConfigureAllowlistModalIdType = 'ADD_ALLOWLIST' | 'EDIT_ALLOWLIST';

type Props = {
    modalId: ConfigureAllowlistModalIdType;
    filterToEdit?: FormValues;
};

const getTitle = (modalId: ConfigureAllowlistModalIdType) => {
    if (modalId === MODAL_TYPE.EDIT_ALLOWLIST) {
        return intl.getMessage('allowlist_edit');
    }

    return intl.getMessage('allowlist_add');
};

const getButtonText = (modalId: ConfigureAllowlistModalIdType) => {
    if (modalId === MODAL_TYPE.EDIT_ALLOWLIST) {
        return intl.getMessage('save');
    }

    return intl.getMessage('add');
};

export const ConfigureAllowlistModal = ({ modalId, filterToEdit }: Props) => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);
    const { processingAddFilter } = filtering;

    const methods = useForm({
        defaultValues: {
            ...defaultValues,
            ...filterToEdit,
        },
        mode: 'onBlur',
    });
    const { handleSubmit, reset, control } = methods;

    useEffect(() => {
        reset({
            ...defaultValues,
            ...filterToEdit,
        });
    }, [filterToEdit, reset]);

    const handleFormSubmit = async (values: FormValues) => {
        switch (modalId) {
            case MODAL_TYPE.ADD_ALLOWLIST: {
                dispatch(addFilter(values.url, values.name, true));
                dispatch(closeModal());
                break;
            }
            case MODAL_TYPE.EDIT_ALLOWLIST: {
                dispatch(editFilter(values.url, values, true));
                dispatch(closeModal());
                break;
            }
            default: {
                break;
            }
        }
    };

    const handleCancel = () => {
        reset(defaultValues);
        dispatch(closeModal());
    };

    return (
        <ModalWrapper id={modalId}>
            <Dialog visible onClose={handleCancel} title={getTitle(modalId)}>
                <FormProvider {...methods}>
                    <form onSubmit={handleSubmit(handleFormSubmit)}>
                        <div>
                            <div className={theme.form.group}>
                                <div className={theme.form.input}>
                                    <Controller
                                        name="name"
                                        control={control}
                                        render={({ field, fieldState }) => (
                                            <Input
                                                {...field}
                                                type="text"
                                                id="filters_name"
                                                label={intl.getMessage('name_label')}
                                                placeholder={intl.getMessage('allowlist_placeholder_example')}
                                                errorMessage={fieldState.error?.message}
                                            />
                                        )}
                                    />
                                </div>

                                <div className={theme.form.input}>
                                    <Controller
                                        name="url"
                                        control={control}
                                        rules={{
                                            validate: { validateRequiredValue, validatePath },
                                        }}
                                        render={({ field, fieldState }) => (
                                            <Input
                                                {...field}
                                                type="text"
                                                id="filters_url"
                                                label={intl.getMessage('blocklist_url_file_path')}
                                                placeholder={intl.getMessage('blocklist_url_file_path')}
                                                errorMessage={fieldState.error?.message}
                                            />
                                        )}
                                    />
                                </div>
                            </div>
                        </div>

                        <div className={theme.dialog.footer}>
                            <Button
                                type="submit"
                                id="filters_save"
                                variant="primary"
                                size="small"
                                disabled={processingAddFilter}
                                className={theme.dialog.button}
                            >
                                {getButtonText(modalId)}
                            </Button>

                            <Button
                                type="button"
                                id="filters_cancel"
                                variant="secondary"
                                size="small"
                                onClick={handleCancel}
                                className={theme.dialog.button}
                            >
                                {intl.getMessage('cancel')}
                            </Button>
                        </div>
                    </form>
                </FormProvider>
            </Dialog>
        </ModalWrapper>
    );
};
