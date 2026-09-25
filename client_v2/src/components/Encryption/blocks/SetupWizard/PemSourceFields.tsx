import { createMemo, Show, untrack } from 'solid-js';

import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { Textarea } from 'panel/common/controls/Textarea';
import { Dropzone } from 'panel/common/ui/Dropzone';
import theme from 'panel/lib/theme';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import { InlineMessage } from './InlineMessage';
import type { PemFields, PemStepConfig } from './pemFields';
import s from './styles.module.pcss';

type Props = {
    config: PemStepConfig;
    fields: PemFields;
    errorFor: (field: string) => string | undefined;
    warningFor: (field: string) => string | undefined;
};

/**
 * The certificate / private key step body: the source switch between pasted
 * PEM data and a path on the server, plus the matching input and dropzone.
 */
export const PemSourceFields = (props: Props) => {
    const config = untrack(() => props.config);
    const fields = untrack(() => props.fields);

    const sourceOptions = createMemo(() => [
        {
            text: config.texts.textOption(),
            description: config.texts.textOptionDesc(),
            value: ENCRYPTION_SOURCE.CONTENT,
        },
        {
            text: config.texts.pathOption(),
            description: config.texts.pathOptionDesc(),
            value: ENCRYPTION_SOURCE.PATH,
        },
    ]);

    const contentWarning = () => props.warningFor(config.fields.content);
    const pathWarning = () => props.warningFor(config.fields.path);

    return (
        <>
            <Radio
                value={fields.sourceValue() ?? ''}
                handleChange={fields.handleSourceChange}
                name={config.radioName}
                options={sourceOptions()}
                optionTestIdPrefix={`${config.testIdPrefix}-source`}
                inModal
            />
            <Show
                when={fields.isContent()}
                fallback={
                    <div class={theme.form.input}>
                        <Input
                            id={`${config.idPrefix}${config.fields.path}`}
                            name={config.fields.path}
                            value={fields.pathValue()}
                            onChange={fields.handlePathChange}
                            onBlur={fields.validateOnBlur}
                            placeholder="/etc/letsencrypt/live/example.com/fullchain.pem"
                            errorMessage={props.errorFor(config.fields.path)}
                            label={config.texts.pathLabel()}
                            size="large"
                            data-testid={`${config.testIdPrefix}-path`}
                        />
                        <Show when={pathWarning()}>
                            <InlineMessage
                                kind="warning"
                                data-testid={`${config.testIdPrefix}-path-warning`}
                            >
                                {pathWarning()}
                            </InlineMessage>
                        </Show>
                    </div>
                }
            >
                <div class={theme.form.input}>
                    <Textarea
                        id={`${config.idPrefix}${config.fields.content}`}
                        name={config.fields.content}
                        value={fields.contentValue()}
                        onChange={fields.handleContentChange}
                        onBlur={fields.validateOnBlur}
                        placeholder={config.contentPlaceholder}
                        errorMessage={props.errorFor(config.fields.content)}
                        label={config.texts.contentLabel()}
                        isClearable
                        onClear={fields.handleClear}
                        size={fields.dropzoneVisible() ? 'compact' : 'large'}
                        data-testid={`${config.testIdPrefix}-content`}
                    />
                    <Show when={contentWarning()}>
                        <InlineMessage
                            kind="warning"
                            data-testid={`${config.testIdPrefix}-content-warning`}
                        >
                            {contentWarning()}
                        </InlineMessage>
                    </Show>
                    <Show when={fields.dropzoneVisible()}>
                        <Dropzone
                            onFileSelect={fields.handleFileSelect}
                            hint={config.texts.dropzoneHint()}
                            testId={`${config.testIdPrefix}-dropzone`}
                            class={s.dropzoneGap}
                        />
                    </Show>
                </div>
            </Show>
        </>
    );
};
