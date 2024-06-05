import React from 'react';
import { Trans, withTranslation } from 'react-i18next';

import ReactModal from 'react-modal';

import { MODAL_TYPE } from '../../../helpers/constants';

import Form from './Form';

const getInitialData = ({ initial, modalType, clientId, clientName }: any) => {
    if (initial && initial.blocked_services) {
        const { blocked_services } = initial;
        const blocked = {};

        blocked_services.forEach((service: any) => {
            blocked[service] = true;
        });

        return {
            ...initial,
            blocked_services: blocked,
        };
    }

    if (modalType !== MODAL_TYPE.EDIT_CLIENT && clientId) {
        return {
            ...initial,
            name: clientName,
            ids: [clientId],
        };
    }

    return initial;
};

interface ModalProps {
    isModalOpen: boolean;
    modalType: string;
    currentClientData: object;
    handleSubmit: (values: any) => void;
    handleClose: (...args: unknown[]) => unknown;
    processingAdding: boolean;
    processingUpdating: boolean;
    tagsOptions: unknown[];
    t: (...args: unknown[]) => string;
    clientId?: string;
}

const Modal = ({
    isModalOpen,
    modalType,
    currentClientData,
    handleSubmit,
    handleClose,
    processingAdding,
    processingUpdating,
    tagsOptions,
    clientId,
    t,
}: ModalProps) => {
    const initialData = getInitialData({
        initial: currentClientData,
        modalType,
        clientId,
        clientName: t('client_name', { id: clientId }),
    });

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered modal-dialog--clients"
            closeTimeoutMS={0}
            isOpen={isModalOpen}
            onRequestClose={handleClose}>
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">
                        {modalType === MODAL_TYPE.EDIT_CLIENT ? <Trans>client_edit</Trans> : <Trans>client_new</Trans>}
                    </h4>

                    <button type="button" className="close" onClick={handleClose}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>

                <Form
                    initialValues={{ ...initialData }}
                    onSubmit={handleSubmit}
                    handleClose={handleClose}
                    processingAdding={processingAdding}
                    processingUpdating={processingUpdating}
                    tagsOptions={tagsOptions}
                />
            </div>
        </ReactModal>
    );
};

export default withTranslation()(Modal);
