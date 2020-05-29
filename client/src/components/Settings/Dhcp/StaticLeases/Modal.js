import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import ReactModal from 'react-modal';

import Form from './Form';

const Modal = (props) => {
    const {
        isModalOpen,
        handleSubmit,
        toggleLeaseModal,
        processingAdding,
    } = props;

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered modal-dialog--clients"
            closeTimeoutMS={0}
            isOpen={isModalOpen}
            onRequestClose={() => toggleLeaseModal()}
        >
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">
                        <Trans>dhcp_new_static_lease</Trans>
                    </h4>
                    <button type="button" className="close" onClick={() => toggleLeaseModal()}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <Form
                    onSubmit={handleSubmit}
                    toggleLeaseModal={toggleLeaseModal}
                    processingAdding={processingAdding}
                />
            </div>
        </ReactModal>
    );
};

Modal.propTypes = {
    isModalOpen: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    toggleLeaseModal: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
};

export default withTranslation()(Modal);
