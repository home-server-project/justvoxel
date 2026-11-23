const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const path = require('node:path');
const { test } = require('node:test');
const vm = require('node:vm');

const apiPath = process.env.JUSTVOXEL_BOOTC_API ||
    path.join(__dirname, '../installer/webui-overrides/src/apis/justvoxel-bootc.js');
const api = readFileSync(apiPath, 'utf8').replace(/^import .*;$/gm, '').replace(/^export /gm, '');

for (const edition of ['hws', 'vm']) {
    test(edition + ' installs from the registry and tracks the stable image', async () => {
        let configuration;
        const client = {
            async call(objectPath, dbusInterface, method, [, property, value]) {
                if (method === 'Set' && property === 'Configuration') {
                    configuration = value.v;
                    return;
                }
                const properties = { Sources: ['/source'], Type: 'BOOTC', Configuration: {} };
                assert.equal(method, 'Get');
                return [{ v: properties[property] }];
            },
        };
        const { JUSTVOXEL_IMAGE_REFS, setJustVoxelEdition } = vm.runInNewContext(
            api + '\n;({ JUSTVOXEL_IMAGE_REFS, setJustVoxelEdition })',
            {
                cockpit: { variant: (type, v) => ({ type, v }) },
                PayloadsClient: { instance: { client } },
                getActivePayload: async () => '/payload',
            }
        );

        const target = 'ghcr.io/home-server-project/justvoxel-' + edition + ':stable';
        const selected = await setJustVoxelEdition(edition);
        assert.doesNotMatch(configuration.targetImgRef.v, /^registry:/);
        assert.equal(configuration.sourceImgRef.v, 'registry:' + target);
        assert.equal(configuration.targetImgRef.v, target);
        assert.equal(JUSTVOXEL_IMAGE_REFS[edition], target);
        assert.equal(selected, target);
    });
}
