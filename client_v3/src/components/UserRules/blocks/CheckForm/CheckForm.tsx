import { createSignal, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import { Button } from 'panel/common/ui/Button';
import theme from 'panel/lib/theme';
import { DNS_RECORD_TYPE_OPTIONS } from '../../types';

import s from './CheckForm.module.pcss';

type Props = {
    hostname: string;
    client: string;
    qtype: string;
    onHostnameChange: (value: string) => void;
    onClientChange: (value: string) => void;
    onQtypeChange: (value: string) => void;
    handleSubmit: () => void | Promise<void>;
    processingCheck: boolean;
};

export const CheckForm = (props: Props) => {
    const [hostnameError, setHostnameError] = createSignal<string | undefined>();
    const [qtypeError, setQtypeError] = createSignal<string | undefined>();

    const isValid = () => !!props.hostname && !!props.qtype && !hostnameError() && !qtypeError();

    const handleSubmit = (e: Event) => {
        e.preventDefault();

        if (!props.hostname) {
            setHostnameError(intl.getMessage('form_error_required'));
            return;
        }
        if (!props.qtype) {
            setQtypeError(intl.getMessage('form_error_required'));
            return;
        }

        props.handleSubmit();
    };

    const handleHostnameChange = (e: Event) => {
        const value = (e.target as HTMLInputElement).value;
        props.onHostnameChange(value);
        setHostnameError(value ? undefined : intl.getMessage('form_error_required'));
    };

    const handleClientChange = (e: Event) => {
        props.onClientChange((e.target as HTMLInputElement).value);
    };

    const handleQtypeChange = (option: { value: string } | null) => {
        const value = option?.value || '';
        props.onQtypeChange(value);
        setQtypeError(value ? undefined : intl.getMessage('form_error_required'));
    };

    return (
        <form onSubmit={handleSubmit}>
            <div class={s.formFields}>
                <div class={s.formGroup}>
                    <Input
                        id="user-rules-hostname"
                        data-testid="user-rules-check-hostname"
                        type="text"
                        size="medium"
                        label={intl.getMessage('user_rules_check_hostname_label')}
                        placeholder={intl.getMessage('user_rules_check_hostname_placeholder')}
                        value={props.hostname}
                        onChange={handleHostnameChange}
                        errorMessage={hostnameError()}
                        isClearable
                        onClear={() => {
                            props.onHostnameChange('');
                            setHostnameError(intl.getMessage('form_error_required'));
                        }}
                    />
                </div>

                <div class={s.formGroup}>
                    <Input
                        id="user-rules-client"
                        data-testid="user-rules-check-client"
                        type="text"
                        size="medium"
                        label={intl.getMessage('user_rules_check_client_label')}
                        placeholder={intl.getMessage('user_rules_check_client_placeholder')}
                        value={props.client}
                        onChange={handleClientChange}
                        isClearable
                        onClear={() => props.onClientChange('')}
                    />
                </div>

                <div class={s.formGroup}>
                    <div class={s.selectField} data-testid="user-rules-check-qtype">
                        <label
                            class={cn(s.selectLabel, theme.text.t3)}
                            for="user-rules-qtype-input"
                        >
                            {intl.getMessage('user_rules_check_dns_record_type_label')}
                        </label>

                        <Select
                            id="user-rules-qtype"
                            inputId="user-rules-qtype-input"
                            size="responsive"
                            height="medium"
                            menuSize="large"
                            placeholder={intl.getMessage('user_rules_dns_record_type_placeholder')}
                            options={DNS_RECORD_TYPE_OPTIONS}
                            value={DNS_RECORD_TYPE_OPTIONS.find(
                                (option) => option.value === props.qtype,
                            )}
                            onChange={handleQtypeChange}
                        />

                        <Show when={qtypeError()}>
                            <div class={theme.form.error}>{qtypeError()}</div>
                        </Show>
                    </div>
                </div>
            </div>

            <div class={s.checkActions}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={!isValid() || props.processingCheck}
                    class={s.checkSubmitButton}
                    data-testid="user-rules-check-submit"
                >
                    {intl.getMessage('user_rules_check_button')}
                </Button>
            </div>
        </form>
    );
};
