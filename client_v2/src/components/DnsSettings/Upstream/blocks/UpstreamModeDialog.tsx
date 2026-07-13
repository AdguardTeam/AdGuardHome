import { createSignal, createEffect, createMemo, type Accessor } from 'solid-js';
import cn from 'clsx';
import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Radio } from 'panel/common/controls/Radio';
import { getUpstreamModeOptions } from '../../helpers';
import theme from 'panel/lib/theme';

import s from './UpstreamModeDialog.module.pcss';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const UpstreamModeDialog = (props: Props) => {
    const [upstreamMode, setUpstreamMode] = createSignal(dnsConfigState.upstream_mode);

    createEffect(() => {
        if (props.open()) {
            setUpstreamMode(dnsConfigState.upstream_mode);
        }
    });

    const upstreamModeOptions = createMemo(() => {
        return getUpstreamModeOptions().map((opt) => ({
            text: opt.text,
            value: opt.value,
            description: opt.warning ? (
                <>
                    {opt.description}
                    <div class={cn(theme.text.t3, s.warning)}>{opt.warning}</div>
                </>
            ) : (
                opt.description
            ),
        }));
    });

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_upstream_mode_title')}
            description={intl.getMessage('dns_upstream_mode_desc')}
            onClose={props.onClose}
            onSubmit={() => {
                setDnsConfig({ upstream_mode: upstreamMode() });
                props.onClose();
            }}
            processing={props.processing}
        >
            <Radio
                name="upstream_mode"
                options={upstreamModeOptions()}
                value={upstreamMode()}
                handleChange={(v: string) => setUpstreamMode(v)}
                inModal
            />
        </ConfigDialog>
    );
};
