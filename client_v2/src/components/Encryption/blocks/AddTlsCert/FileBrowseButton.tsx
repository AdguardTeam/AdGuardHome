import intl from 'panel/common/intl';
import s from './styles.module.pcss';

type Props = {
    onFileSelect: (content: string) => void;
};

export const FileBrowseButton = (props: Props) => {
    let fileInputRef: HTMLInputElement | undefined;

    const handleClick = () => {
        fileInputRef?.click();
    };

    const handleFileChange = (e: Event) => {
        const input = e.currentTarget as HTMLInputElement;
        const file = input.files?.[0];
        if (!file) return;

        const reader = new FileReader();
        reader.onload = () => {
            const content = reader.result as string;
            props.onFileSelect(content);
        };
        reader.readAsText(file);

        // Reset so the same file can be selected again.
        input.value = '';
    };

    return (
        <>
            <button type="button" class={s.browseLink} onClick={handleClick}>
                {intl.getMessage('browse')}
            </button>
            <input
                ref={(el) => (fileInputRef = el)}
                type="file"
                class={s.hiddenFileInput}
                onChange={handleFileChange}
                aria-hidden="true"
            />
        </>
    );
};
