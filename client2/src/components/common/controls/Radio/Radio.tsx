import React, { FC } from 'react';
import { Radio } from 'antd';
import { observer } from 'mobx-react-lite';

import s from './Radio.module.pcss';

const { Group } = Radio;

interface RadioProps {
    options: {
        label: string;
        desc?: string;
        value: string | number;
    }[];
    onSelect: (value: string | number) => void;
    value: string | number;
}

const RadioComponent: FC<RadioProps> = observer(({
    options, onSelect, value,
}) => {
    if (options.length === 0) {
        return null;
    }

    return (
        <Group
            value={value}
            onChange={(e) => {
                onSelect(e.target.value);
            }}
            className={s.group}
        >
            {options.map((o) => (
                <Radio
                    key={o.value}
                    value={o.value}
                    className={s.radio}
                >
                    <div>
                        {o.label}
                    </div>
                    {o.desc && (
                        <div className={s.desc}>
                            {o.desc}
                        </div>
                    )}
                </Radio>
            ))}
        </Group>

    );
});

export default RadioComponent;
