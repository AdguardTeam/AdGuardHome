import React, { ReactNode } from 'react';
import { useSelector } from 'react-redux';
import { RootState } from 'panel/initialState';

interface Props {
    id: string;
    children: ReactNode;
}

export const ModalWrapper = ({ id, children }: Props) => {
    const modalId = useSelector((state: RootState) => state.modals.modalId);

    if (modalId !== id) {
        return null;
    }

    return <>{children}</>;
};
