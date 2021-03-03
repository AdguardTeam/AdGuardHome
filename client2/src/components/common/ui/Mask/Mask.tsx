import React, { FC } from 'react';
import cn from 'classnames';

import s from './Mask.module.pcss';

interface MaskProps {
    open: boolean;
    handle: () => void;
}

const Mask: FC<MaskProps> = ({ open, handle }) => {
    return (
        <div
            className={cn(
                s.mask,
                { [s.mask_visible]: open },
            )}
            onClick={handle}
        />
    );
};

export default Mask;
