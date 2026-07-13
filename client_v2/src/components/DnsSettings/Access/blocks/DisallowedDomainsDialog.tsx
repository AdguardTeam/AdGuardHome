import { untrack, type Accessor } from 'solid-js';

import { accessState, setAccessList } from 'panel/stores/access';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Textarea } from 'panel/common/controls/Textarea';
import { validateDomainsPerLine } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const DisallowedDomainsDialog = (props: Props) => {
    const field = useField<string>(
        () => props.open(),
        () => accessState.blocked_hosts,
        { validate: (v) => (v ? validateDomainsPerLine(v) || '' : '') },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_disallowed_domains')}
            description={
                <>
                    <p>{intl.getMessage('dns_disallowed_domains_desc')}</p>
                    <p>{intl.getMessage('dns_disallowed_domains_desc_2')}</p>
                </>
            }
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setAccessList({ blocked_hosts: v });
                    untrack(() => props.onClose());
                });
            }}
            processing={props.processing}
        >
            <div class={theme.form.input}>
                <Textarea
                    value={field.value()}
                    onChange={(e) => field.setValue(e.target.value)}
                    onBlur={() => field.validate()}
                    id="blocked_hosts"
                    label={intl.getMessage('dns_disallowed_domains_label')}
                    size="medium"
                    errorMessage={field.error()}
                />
            </div>
        </ConfigDialog>
    );
};
