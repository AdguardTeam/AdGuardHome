import { createSignal, createEffect, type Accessor } from 'solid-js';

import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { EDNS_MODES } from 'panel/helpers/constants';
import { getEdnsOptions } from '../../helpers';
import { validateRequiredValue, validateIp } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const EdnsDialog = (props: Props) => {
    const ednsOptions = getEdnsOptions();

    // EDNS custom flag — plain signal synced from store
    const [ednsCsUseCustom, setEdnsCsUseCustom] = createSignal(
        dnsConfigState.edns_cs_use_custom ?? false,
    );
    createEffect(() => {
        if (props.open()) {
            setEdnsCsUseCustom(dnsConfigState.edns_cs_use_custom ?? false);
        }
    });

    // Custom IP field — conditionally validated
    const ednsCsCustomIp = useField<string>(
        () => props.open(),
        () => dnsConfigState.edns_cs_custom_ip ?? '',
        {
            validate: (v) =>
                ednsCsUseCustom() ? validateRequiredValue(v) || validateIp(v) || '' : '',
        },
    );

    // Clear custom IP error when switching away from custom mode
    createEffect(() => {
        if (!ednsCsUseCustom()) {
            ednsCsCustomIp.setError('');
        }
    });

    const handleSubmit = () => {
        // Bugfix: submit-time validation
        if (ednsCsUseCustom() && ednsCsCustomIp.validate()) return;

        const payload: Record<string, unknown> = {
            edns_cs_use_custom: ednsCsUseCustom(),
        };
        if (ednsCsUseCustom()) {
            payload.edns_cs_custom_ip = ednsCsCustomIp.value();
        }
        setDnsConfig(payload);
        props.onClose();
    };

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_edns_title')}
            description={intl.getMessage('dns_edns_desc')}
            onClose={props.onClose}
            onSubmit={handleSubmit}
            processing={props.processing}
        >
            <div class={theme.form.input}>
                <Radio
                    name="edns_cs_mode"
                    options={ednsOptions}
                    value={ednsCsUseCustom() ? EDNS_MODES.custom : EDNS_MODES.default}
                    handleChange={(v: string) => setEdnsCsUseCustom(v === EDNS_MODES.custom)}
                    inModal
                />
                <Input
                    id="edns_cs_custom_ip"
                    label={intl.getMessage('dns_edns_custom_label')}
                    placeholder={intl.getMessage('dns_edns_custom_placeholder')}
                    value={ednsCsCustomIp.value()}
                    onChange={(e: Event) =>
                        ednsCsCustomIp.setValue((e.target as HTMLInputElement).value)
                    }
                    onBlur={() => ednsCsCustomIp.validate()}
                    disabled={!ednsCsUseCustom()}
                    errorMessage={ednsCsCustomIp.error()}
                    size="large"
                />
            </div>
        </ConfigDialog>
    );
};
