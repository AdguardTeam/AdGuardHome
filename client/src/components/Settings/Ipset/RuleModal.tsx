import React, { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

import { Input } from '../../ui/Controls/Input';
import { validateDomainsInput, validateIPSetsInput, formatIPSetRule } from '../../../helpers/ipset';

interface RuleModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSave: (rule: string) => void;
    initialDomains?: string;
    initialIPSets?: string;
    title: string;
}

interface FormData {
    domains: string;
    ipsets: string;
}

const RuleModal: React.FC<RuleModalProps> = ({
    isOpen,
    onClose,
    onSave,
    initialDomains = '',
    initialIPSets = '',
    title,
}) => {
    const { t } = useTranslation();

    const {
        handleSubmit,
        control,
        reset,
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: {
            domains: initialDomains,
            ipsets: initialIPSets,
        },
    });

    // Update form values when props change
    useEffect(() => {
        if (isOpen) {
            reset({
                domains: initialDomains,
                ipsets: initialIPSets,
            });
        }
    }, [isOpen, initialDomains, initialIPSets, reset]);

    const onSubmit = (data: FormData) => {
        const rule = formatIPSetRule({
            domains: data.domains.split(',').map((d) => d.trim()),
            ipsets: data.ipsets.split(',').map((s) => s.trim()),
        });
        onSave(rule);
        reset();
        onClose();
    };

    const handleFormSubmit = (e: React.FormEvent) => {
        // Prevent the form submission from bubbling up to parent form
        e.stopPropagation();
        handleSubmit(onSubmit)(e);
    };

    const handleClose = () => {
        reset();
        onClose();
    };

    const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
        // Only close if clicking on the backdrop itself, not on modal content
        if (e.target === e.currentTarget) {
            handleClose();
        }
    };

    if (!isOpen) {
        return null;
    }

    return (
        <>
            <div className="modal-backdrop fade show"></div>
            <div
                className="modal fade show d-block"
                tabIndex={-1}
                role="dialog"
                onClick={handleBackdropClick}
                style={{ zIndex: 1050 }}
            >
                <div
                    className="modal-dialog modal-dialog-centered"
                    role="document"
                    onClick={(e) => e.stopPropagation()}
                >
                    <div className="modal-content">
                    <div className="modal-header">
                        <h5 className="modal-title">{title}</h5>
                        <button
                            type="button"
                            className="close"
                            onClick={handleClose}>
                            <span className="sr-only">Close</span>
                        </button>
                    </div>
                    <form onSubmit={handleFormSubmit}>
                        <div className="modal-body">
                            <div className="form-group">
                                <Controller
                                    name="domains"
                                    control={control}
                                    rules={{ validate: validateDomainsInput }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            label={t('ipset_domains')}
                                            desc={t('ipset_domains_desc')}
                                            placeholder="example.com,*.example.org"
                                            error={fieldState.error?.message}
                                        />
                                    )}
                                />
                            </div>

                            <div className="form-group">
                                <Controller
                                    name="ipsets"
                                    control={control}
                                    rules={{ validate: validateIPSetsInput }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            label={t('ipset_names')}
                                            desc={t('ipset_names_desc')}
                                            placeholder="my_ipset,another_set"
                                            error={fieldState.error?.message}
                                        />
                                    )}
                                />
                            </div>

                            <div className="alert alert-info">
                                <div className="mb-2">
                                    <strong>{t('ipset_example')}:</strong>
                                </div>
                                <code>example.com,*.example.org/my_ipset,blocked_ips</code>
                            </div>
                        </div>
                        <div className="modal-footer">
                            <button type="button" className="btn btn-secondary" onClick={handleClose}>
                                {t('cancel_btn')}
                            </button>
                            <button type="submit" className="btn btn-success">
                                {t('save_btn')}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
        </>
    );
};

export default RuleModal;
