import { createSignal, Show } from 'solid-js';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import intl from 'panel/common/intl';

import s from './Dropzone.module.pcss';
import theme from 'panel/lib/theme';

export type DropzoneProps = {
    onFileSelect: (content: string) => void;
    hint: string;
    testId: string;
    class?: string;
};

export const Dropzone = (props: DropzoneProps) => {
    let fileInputRef: HTMLInputElement | undefined;
    const [dragOver, setDragOver] = createSignal(false);

    const readFile = (file: File | undefined) => {
        if (!file) return;

        const reader = new FileReader();
        reader.onload = () => {
            props.onFileSelect(reader.result as string);
        };
        reader.readAsText(file);
    };

    const handleClick = () => {
        fileInputRef?.click();
    };

    const handleFileChange = (e: Event) => {
        const input = e.currentTarget as HTMLInputElement;
        readFile(input.files?.[0]);
        input.value = '';
    };

    const handleDragOver = (e: DragEvent) => {
        e.preventDefault();
        setDragOver(true);
    };

    const handleDragLeave = (e: DragEvent) => {
        // Ignore the dragleave fired when the pointer moves onto a child node.
        const next = e.relatedTarget as Node | null;
        if (next && e.currentTarget instanceof Node && e.currentTarget.contains(next)) {
            return;
        }
        setDragOver(false);
    };

    const handleDrop = (e: DragEvent) => {
        e.preventDefault();
        setDragOver(false);
        readFile(e.dataTransfer?.files?.[0]);
    };

    return (
        <>
            <button
                type="button"
                class={cn(s.dropzone, theme.text.t3, props.class, { [s.dragOver]: dragOver() })}
                data-testid={props.testId}
                onClick={handleClick}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
            >
                <span>{props.hint}</span>
                <Show
                    when={dragOver()}
                    fallback={<span class={s.browseLink}>{intl.getMessage('browse')}</span>}
                >
                    <Icon icon="download" class={s.dropzoneIcon} />
                </Show>
            </button>
            <input
                ref={(el) => (fileInputRef = el)}
                type="file"
                class={s.hiddenFileInput}
                onChange={handleFileChange}
                tabindex="-1"
                aria-hidden="true"
            />
        </>
    );
};
