import { createSignal, createEffect, untrack } from 'solid-js';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { MODAL_TYPE } from 'panel/helpers/constants';

import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { closeModal } from 'panel/stores/modals';
import theme from 'panel/lib/theme';
import { Button } from 'panel/common/ui/Button';
import { addFilter, editFilter, filteringState } from 'panel/stores/filtering';
import { Input } from 'panel/common/controls/Input';
import { validatePath, validateRequiredValue } from 'panel/helpers/validators';

type FormValues = {
    name: string;
    url: string;
    enabled?: boolean;
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

export const ConfigureAllowlistModal = (props: Props) => {
    const [name, setName] = createSignal(untrack(() => props.filterToEdit?.name) ?? '');
    const [url, setUrl] = createSignal(untrack(() => props.filterToEdit?.url) ?? '');
    const [urlError, setUrlError] = createSignal<string | undefined>();

    createEffect(() => {
        setName(props.filterToEdit?.name ?? '');
        setUrl(props.filterToEdit?.url ?? '');
    });

    const validateAndSetErrors = () => {
        const urlErr = validateRequiredValue(url()) || validatePath(url());
        setUrlError(urlErr || undefined);
        return !urlErr;
    };

    const handleFormSubmit = async (e: Event) => {
        e.preventDefault();

        if (!validateAndSetErrors()) {
            return;
        }

        const values: FormValues = { name: name(), url: url() };

        switch (props.modalId) {
            case MODAL_TYPE.ADD_ALLOWLIST: {
                addFilter(values.url, values.name, true);
                break;
            }
            case MODAL_TYPE.EDIT_ALLOWLIST: {
                editFilter(props.filterToEdit!.url, values, true);
                break;
            }
            default: {
                break;
            }
        }

        setName('');
        setUrl('');
        closeModal();
    };

    const handleCancel = () => {
        setName('');
        setUrl('');
        setUrlError(undefined);
        closeModal();
    };

    return (
        <ModalWrapper id={props.modalId}>
            <Dialog visible onClose={handleCancel} title={getTitle(props.modalId)}>
                <form onSubmit={handleFormSubmit}>
                    <div>
                        <div class={theme.form.group}>
                            <div class={theme.form.input}>
                                <Input
                                    type="text"
                                    id="filters_name"
                                    label={intl.getMessage('name_label')}
                                    placeholder={intl.getMessage('allowlist_placeholder_example')}
                                    value={name()}
                                    onChange={(e) => setName((e.target as HTMLInputElement).value)}
                                />
                            </div>

                            <div class={theme.form.input}>
                                <Input
                                    type="text"
                                    id="filters_url"
                                    label={intl.getMessage('blocklist_url_file_path')}
                                    placeholder={intl.getMessage('blocklist_url_file_path')}
                                    value={url()}
                                    onChange={(e) => setUrl((e.target as HTMLInputElement).value)}
                                    onBlur={validateAndSetErrors}
                                    errorMessage={urlError()}
                                />
                            </div>
                        </div>
                    </div>

                    <div class={theme.dialog.footer}>
                        <Button
                            type="submit"
                            id="filters_save"
                            variant="primary"
                            size="small"
                            disabled={filteringState.processingAddFilter}
                            class={theme.dialog.button}
                        >
                            {getButtonText(props.modalId)}
                        </Button>

                        <Button
                            type="button"
                            id="filters_cancel"
                            variant="secondary"
                            size="small"
                            onClick={handleCancel}
                            class={theme.dialog.button}
                        >
                            {intl.getMessage('cancel')}
                        </Button>
                    </div>
                </form>
            </Dialog>
        </ModalWrapper>
    );
};
