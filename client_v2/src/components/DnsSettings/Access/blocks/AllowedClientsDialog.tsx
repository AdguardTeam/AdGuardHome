import { untrack, type Accessor } from 'solid-js';

import { accessState, setAccessList } from 'panel/stores/access';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Textarea } from 'panel/common/controls/Textarea';
import { validateClientsPerLine } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const AllowedClientsDialog = (props: Props) => {
    const field = useField<string>(
        () => props.open(),
        () => accessState.allowed_clients,
        { validate: (v) => (v ? validateClientsPerLine(v) || '' : '') },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_allowed_clients')}
            description={
                <>
                    <p>{intl.getMessage('dns_allowed_clients_desc')}</p>
                    <p>
                        {intl.getMessage('dns_allowed_clients_desc_2', {
                            a: (text: string) => (
                                <a
                                    href="https://github.com/AdguardTeam/AdGuardHome/wiki/Clients#identifying-clients"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    class={theme.link.link}
                                >
                                    {text}
                                </a>
                            ),
                        })}
                    </p>
                </>
            }
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setAccessList({ allowed_clients: v });
                    untrack(() => props.onClose());
                });
            }}
            processing={props.processing}
        >
            <div class={theme.form.input}>
                <Textarea
                    value={field.value()}
                    onChange={(e: Event) => field.setValue((e.target as HTMLTextAreaElement).value)}
                    onBlur={() => field.validate()}
                    id="allowed_clients"
                    label={intl.getMessage('dns_allowed_clients_label')}
                    placeholder={intl.getMessage('dns_allowed_clients_placeholder')}
                    size="medium"
                    errorMessage={field.error()}
                />
            </div>
        </ConfigDialog>
    );
};
