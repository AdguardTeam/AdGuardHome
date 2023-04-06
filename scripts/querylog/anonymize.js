const fs = require('fs');
const readline = require('readline');
const dnsPacket = require('dns-packet')

const processLineByLine = async (source, callback) => {
    const fileStream = fs.createReadStream(source);

    const rl = readline.createInterface({
        input: fileStream,
        crlfDelay: Infinity
    });

    for await (const line of rl) {
        await callback(line);
    }
}

const anonDomain = (domain) => {
    // Replace all question domain letters with a
    return domain.replace(/[a-z]/g, 'a');
}

const anonIP = (ip) => {
    // Replace all numbers with '1'
    return ip.replace(/[0-9]/g, '1');
}

const anonAnswer = (answer) => {
    const answerData = Buffer.from(answer, 'base64');
    const packet = dnsPacket.decode(answerData, 0);

    packet.questions.forEach((q) => {
        q.name = anonDomain(q.name);
    });
    packet.answers.forEach((q) => {
        q.name = anonDomain(q.name);

        if (q.type === 'A' || q.type === 'AAAA') {
            q.data = anonIP(q.data);
        } else if (typeof q.data === 'string') {
            q.data = anonDomain(q.data);
        }
    });

    const anonData = dnsPacket.encode(packet);
    return anonData.toString('base64');
}

const anonLine = (line) => {
    if (!line) {
        return null;
    }

    try {
        const logItem = JSON.parse(line);

        // Replace all numbers with '1'
        logItem['IP'] = logItem['IP'].replace(/[0-9]/g, '1');
        // Replace all question domain letters with a
        logItem['QH'] = logItem['QH'].replace(/[a-z]/g, 'a');
        // Anonymize "Answer" and "OrigAnswer" fields
        if (logItem['Answer']) {
            logItem['Answer'] = anonAnswer(logItem['Answer']);
        }
        if (logItem['OrigAnswer']) {
            logItem['OrigAnswer'] = anonAnswer(logItem['OrigAnswer']);
        }

        // If Result is set, anonymize the "Rule" field
        if (logItem['Result'] && logItem['Result']['Rule']) {
            logItem['Result']['Rule'] = anonDomain(logItem['Result']['Rule']);
        }

        return JSON.stringify(logItem);
    } catch (ex) {
        console.error(`Failed to parse ${line}: ${ex} ${ex.stack}`);
        return null;
    }
}

const anon = async (source, dest) => {
    const out = fs.createWriteStream(dest, {
        flags: 'w',
    });


    await processLineByLine(source, async (line) => {
        const newLine = anonLine(line);
        if (!newLine) {
            return;
        }
        out.write(`${newLine}\n`);
    });
}

const main = async () => {
    console.log('Start query log anonymization');

    const source = process.argv[2];
    const dest = process.argv[3];

    console.log(`Source: ${source}`);
    console.log(`Destination: ${dest}`);

    if (!fs.existsSync(source)) {
        throw new Error(`${source} not found`);
    }

    try {
        await anon(source, dest);
    } catch (ex) {
        console.error(ex);
    }

    console.log('Finished query log anonymization')
}

main();
