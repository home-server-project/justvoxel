#!/usr/bin/bash
# Isolated real-file fixtures; also runnable with podman unshare by a human.
# No real Minecraft, root system files, network mounts, or service mutations.
set -euo pipefail
export JV_BACKUP_TEST_REPO="$(cd -- "$(dirname -- "$0")/.." && pwd)"
python3 - <<'PY'
import gzip, hashlib, io, json, os, struct, subprocess, tarfile, unittest
from pathlib import Path

REPO = Path(os.environ['JV_BACKUP_TEST_REPO'])
# Reuse the Phase 4A fixture's actual metadata writer and mount validation.
text = (REPO/'tests/test-instance-backups.sh').read_text().split("python3 - <<'PY'\n",1)[1].rsplit('\nPY',1)[0]
namespace = {'__name__':'backup_fixtures'}
exec(compile(text.replace('unittest.main()',''),'<backup fixtures>','exec'),namespace)
Base = namespace['InstanceBackups']

def world_metadata(version='1.21.4', data_version=4189, extra_data=b''):
    def name(value):
        raw=value.encode();return struct.pack('>H',len(raw))+raw
    raw=b'\x0a'+name('')+b'\x0a'+name('Data')
    raw+=b'\x03'+name('DataVersion')+struct.pack('>i',data_version)
    raw+=b'\x0a'+name('Version')+b'\x03'+name('Id')+struct.pack('>i',data_version)
    raw+=b'\x08'+name('Name')+name(version)+b'\x01'+name('Snapshot')+b'\x00'
    return gzip.compress(raw+b'\x00'+extra_data+b'\x00\x00', mtime=0)

class IndependentRestore(unittest.TestCase):
    private = Base.private
    tool = Base.tool
    config = Base.config
    run_helper = Base.run_helper
    inventory = Base.inventory
    pin_backup = Base.pin_backup

    def test_restore_uses_persisted_backup_namespace(self):
        selected=self.root/'selected-backups'
        self.pin_backup('jv-aaaaaa',selected)
        (self.data/'world/level.dat').write_bytes(world_metadata())
        self.run_helper('create')
        item=self.inventory()[0]
        self.archive=item['archive']
        self.config(network=True)
        self.env['FIXTURE_MOUNT_ID']='disconnected-global-default'
        result=self.plan()
        self.assertEqual(result['metadata']['id'],'jv-aaaaaa')
        self.assertTrue((selected/'jv-aaaaaa'/self.archive).exists())

    def setUp(self):
        Base.setUp(self)
        self.assertFalse((self.registry/'minecraft.json').exists())
        self.restore_source = (REPO/'mjust/libexec/instance-restore').read_text()
        self.assertNotIn('minecraft.json', self.restore_source)
        for old,new in [('/usr/libexec/justvoxel/mjust/',str(self.helpers)+'/'),('/etc/justvoxel/instances',str(self.configs)),('/var/lib/justvoxel/instances',str(self.registry)),('/etc/justvoxel/minecraft-backup.env',str(self.root/'backup.env'))]:
            self.restore_source=self.restore_source.replace(old,new)
        self.restore_source=self.restore_source.replace(
            "try:\n    MANAGEMENT_GID = grp.getgrnam",
            "def fixture_getgrnam(name):\n"
            "    assert name == 'justvoxel-web'\n"
            "    return type('Group', (), {'gr_gid': int(os.environ['FIXTURE_MANAGEMENT_GID'])})()\n"
            "grp.getgrnam = fixture_getgrnam\n"
            "try:\n    MANAGEMENT_GID = grp.getgrnam")
        self.env['FIXTURE_MANAGEMENT_GID']='24027'
        world=self.root/'storage/minecraft-instances/jv-aaaaaa/data/world'
        world.mkdir();(world/'level.dat').write_bytes(world_metadata())
        self.run_helper('create')
        self.item=self.inventory()[0]
        self.archive=self.item['archive']
        self.data=self.root/'storage/minecraft-instances/jv-aaaaaa/data'
        self.other=self.root/'storage/minecraft-instances/jv-bbbbbb/data/level.dat'
        (self.data/'level.dat').write_bytes(b'original at restore time')

    def restore(self,action,plan='',ok=True,identity='jv-aaaaaa',source=None,archive=None,clone_source=None,initialization=None):
        helper=self.helpers/'restore-worker';self.tool(helper,source or self.restore_source)
        args=['bash',str(helper),action,identity,self.archive if archive is None else archive,plan,'false']
        if clone_source is not None: args.append(clone_source)
        if initialization is not None: args.append(initialization)
        result=subprocess.run(args,env=self.env,capture_output=True,text=True,timeout=30)
        self.assertEqual(result.returncode==0,ok,result.stdout+result.stderr)
        self.assertNotIn('private-runtime-secret',result.stdout+result.stderr)
        return result

    def plan(self,**kw): return json.loads(self.restore('plan',**kw).stdout)
    def journal(self): return json.loads((self.configs/'jv-aaaaaa/restore-journal.json').read_text())
    def tx(self): return self.data.parent/('.restore-'+self.journal()['transaction'])
    def prepare(self):
        plan=self.plan();self.restore('prepare',plan['fingerprint']);return plan
    def replace_archive(self,members=None,raw=None):
        folder=self.backups/'jv-aaaaaa'; path=folder/self.archive
        if raw is None:
            with tarfile.open(path,'w:gz') as tf:
                root=tarfile.TarInfo('data');root.type=tarfile.DIRTYPE;root.mode=0o750;tf.addfile(root)
                world=tarfile.TarInfo('data/world/level.dat');world.mode=0o640
                content=world_metadata();world.size=len(content);tf.addfile(world,io.BytesIO(content))
                for name,kind,value in members:
                    m=tarfile.TarInfo(name);m.mode=0o640
                    if kind=='file':
                        m.size=len(value);tf.addfile(m,io.BytesIO(value))
                    elif kind in ('symlink','hardlink'):
                        m.type=tarfile.SYMTYPE if kind=='symlink' else tarfile.LNKTYPE;m.linkname=value;tf.addfile(m)
                    elif kind=='fifo':
                        m.type=tarfile.FIFOTYPE;tf.addfile(m)
        else:path.write_bytes(raw)
        m=json.loads((folder/(self.archive+'.meta.json')).read_text())
        m['size']=path.stat().st_size;m['sha256']=hashlib.sha256(path.read_bytes()).hexdigest()
        self.private(folder/(self.archive+'.meta.json'),json.dumps(m))

    def bedrock_backup(self):
        import shutil
        shutil.rmtree(self.data/'world')
        (self.data/'level.dat').unlink()
        world=self.data/'worlds/Bedrock level';world.mkdir(parents=True)
        payload=b'\x0a\x00\x00\x00'
        (world/'level.dat').write_bytes(struct.pack('<II',10,len(payload))+payload)
        (world/'db').mkdir();(world/'db/CURRENT').write_bytes(b'world database')
        path=self.configs/'jv-aaaaaa/config.json';record=json.loads(path.read_text())
        record['runtime'].update(software='bedrock',version='1.26.50.1')
        self.private(path,json.dumps(record));self.run_helper('create')
        item=next(m for m in self.inventory() if m['software']=='bedrock')
        self.archive=item['archive']
        return world,path,record

    def test_bedrock_exact_same_instance_restore_and_rollback(self):
        world,path,record=self.bedrock_backup()
        original=(world/'db/CURRENT').read_bytes()
        (world/'db/CURRENT').write_bytes(b'newer original data')
        plan=self.prepare()
        self.assertEqual(plan['metadata']['software'],'bedrock')
        self.assertEqual(plan['metadata']['version'],'1.26.50.1')
        self.restore('shutdown');self.restore('activate')
        self.assertEqual((world/'db/CURRENT').read_bytes(),original)
        self.restore('rollback');self.restore('rollback-verifying');self.restore('rolled-back')
        self.assertEqual((world/'db/CURRENT').read_bytes(),b'newer original data')
        self.assertEqual(json.loads(path.read_text())['runtime']['id'],'jv-aaaaaa')

    def test_bedrock_restore_rejects_java_and_wrong_exact_version(self):
        world,path,record=self.bedrock_backup()
        for software,version in [('paper','1.26.50.1'),('bedrock','1.26.51.1')]:
            record['runtime'].update(software=software,version=version)
            self.private(path,json.dumps(record));self.restore('plan',ok=False)
        record['runtime'].update(software='bedrock',version='1.26.50.1')
        self.private(path,json.dumps(record))
        self.assertEqual(self.plan()['integrity'],'verified')
        # A Java archive can never become a Bedrock archive via destination selection.
        self.archive=self.item['archive'];self.restore('plan',ok=False)

    def test_mod_loader_pin_required_for_restore(self):
        config=self.configs/'jv-aaaaaa/config.json'
        record=json.loads(config.read_text())
        record['runtime']['software']='fabric'
        record['runtime']['loader_version']='0.16.10'
        self.private(config,json.dumps(record))
        self.run_helper('create')
        item=next(m for m in self.inventory() if m['software']=='fabric')
        self.archive=item['archive']
        self.assertEqual(self.plan()['metadata']['loader_version'],'0.16.10')
        record['runtime']['loader_version']='0.16.11'
        self.private(config,json.dumps(record))
        self.restore('plan',ok=False)
        self.assertEqual((self.data/'world/level.dat').read_bytes(),world_metadata())

    def test_neoforge_to_quilt_real_archive_and_retained_restore(self):
        source, clone = 'jv-aaaaaa', 'jv-bbbbbb'
        source_config = self.configs/source/'config.json'
        clone_config = self.configs/clone/'config.json'
        source_record = json.loads(source_config.read_text())
        clone_record = json.loads(clone_config.read_text())
        source_record['runtime'].update(software='neoforge', loader_version='21.4.147', bedrock_enabled=True, bedrock_port=19132)
        clone_record['runtime'].update(software='quilt', loader_version='0.28.1', bedrock_enabled=False, bedrock_port=0, start_on_boot=False, preserve_content=True)
        self.private(source_config,json.dumps(source_record))
        self.private(clone_config,json.dumps(clone_record))
        marker = self.data/'world/region/r.0.0.mca'
        marker.parent.mkdir()
        marker.write_bytes(b'Switch archive region marker')
        self.run_helper('create',source)
        archive = next(m for m in self.inventory(source) if m['software']=='neoforge')
        self.archive = archive['archive']
        args = {'identity': clone, 'clone_source': source}
        plan = self.plan(**args)
        self.assertEqual(plan['metadata'],archive)
        self.assertEqual((plan['current_software'],plan['current_version']),('quilt','1.21.4'))
        self.restore('prepare',plan['fingerprint'],**args)
        self.restore('shutdown',identity=clone)
        self.restore('activate',identity=clone)
        clone_data = self.other.parent
        self.assertEqual((clone_data/'world/region/r.0.0.mca').read_bytes(),marker.read_bytes())
        self.restore('rollback',identity=clone)
        self.restore('rollback-verifying',identity=clone)
        self.restore('rolled-back',identity=clone)
        self.assertEqual(self.other.read_bytes(),b'jv-bbbbbb world')
        self.assertFalse((clone_data/'world').exists())
        self.assertEqual(marker.read_bytes(),b'Switch archive region marker')
        self.assertEqual(json.loads(source_config.read_text()),source_record)

    def test_neoforge_to_paper_clone_stages_source_owned_exact_version_archive(self):
        source, clone = 'jv-aaaaaa', 'jv-bbbbbb'
        source_config = self.configs/source/'config.json'
        clone_config = self.configs/clone/'config.json'
        source_record = json.loads(source_config.read_text())
        source_record['runtime'].update(software='neoforge', loader_version='21.4.147', version_policy='pinned')
        self.private(source_config,json.dumps(source_record))
        clone_record = json.loads(clone_config.read_text())
        clone_record['runtime'].update(start_on_boot=False, version_policy='pinned', preserve_content=True)
        self.private(clone_config,json.dumps(clone_record))
        self.assertEqual(source_record['storage']['root'],clone_record['storage']['root'])
        self.run_helper('create',source)
        ordinary = next(m for m in self.inventory(source) if m['software']=='neoforge')
        self.assertEqual(self.plan(identity=source,archive=ordinary['archive'])['metadata'],ordinary)
        # A deployed source can still use the global backup configuration,
        # while a new clone gets an explicit placement from the snapshot.
        self.pin_backup(clone,self.root/'clone-backups')
        (self.root/'clone-backups'/clone).mkdir()

        # A Java-written mod string with a NUL and supplementary character.
        # NBT uses modified UTF-8, including UTF-16 surrogate code units.
        mod_data = b'\x08\x00\x07ModNote\x00\x08\xc0\x80\xed\xa0\xbd\xed\xb8\x80'
        world = world_metadata(extra_data=mod_data)
        (self.data/'world/level.dat').write_bytes(world)
        self.run_helper('create',source)
        item = next(m for m in self.inventory(source) if m['software']=='neoforge' and m['archive']!=ordinary['archive'])
        self.archive = item['archive']
        args = {'identity': clone, 'clone_source': source}
        plan = self.plan(**args)
        self.assertEqual(plan['metadata'],item)
        self.assertEqual((plan['metadata']['id'],plan['metadata']['loader_version']),(source,'21.4.147'))
        self.assertEqual((plan['current_software'],plan['current_version']),('paper','1.21.4'))

        # Even in the clone's own backup namespace, foreign ownership fails.
        clone_backups = self.root/'clone-backups'/clone
        meta = self.backups/source/(self.archive+'.meta.json')
        self.private(clone_backups/meta.name,meta.read_text())
        self.restore('plan',identity=clone,ok=False)
        owned = dict(item,id=clone)
        self.private(clone_backups/meta.name,json.dumps(owned))
        result = self.restore('plan',identity=clone,ok=False)
        self.assertEqual(json.loads(result.stdout),{'error':'incompatible_software_version'})

        for patch, code in [
            ({'id': clone}, 'invalid_metadata_binding'),
            ({'configuration_sha256': 'a'*64}, 'invalid_metadata_binding'),
            ({'loader_version': '21.4.148'}, 'invalid_loader_evidence'),
            ({'loader_version': '/private/archive-controlled'}, 'invalid_loader_evidence'),
            ({'version': '1.21.3'}, 'incompatible_software_version'),
            ({'sha256': 'a'*64}, 'invalid_archive_checksum'),
        ]:
            with self.subTest(patch=patch):
                self.private(meta,json.dumps(dict(item,**patch)))
                result = self.restore('plan',ok=False,**args)
                self.assertEqual(json.loads(result.stdout),{'error':code})
                self.assertNotIn(str(self.root),result.stdout+result.stderr)
                self.assertNotIn('archive-controlled',result.stdout+result.stderr)
        self.private(meta,json.dumps(item))
        for invalid_source in (clone,'/private/archive-controlled'):
            result = self.restore('plan',identity=clone,clone_source=invalid_source,ok=False)
            self.assertEqual(json.loads(result.stdout),{'error':'invalid_clone_identity'})
        result = self.restore('prepare','0'*64,ok=False,**args)
        self.assertEqual(json.loads(result.stdout),{'error':'stale_plan'})
        self.restore('prepare',plan['fingerprint'],**args)
        result = self.restore('plan',ok=False,**args)
        self.assertEqual(json.loads(result.stdout),{'error':'recovery_required'})
        # Only planning and preparation carry clone_source. Durable activation
        # and rollback retain the independent restore journal's destination ID.
        self.restore('shutdown',identity=clone)
        self.restore('activate',identity=clone)
        self.assertEqual((self.other.parent/'world/level.dat').read_bytes(),world)
        self.assertEqual((self.data/'world/level.dat').read_bytes(),world)
        self.restore('rollback',identity=clone)
        self.restore('rollback-verifying',identity=clone)
        self.restore('rolled-back',identity=clone)
        self.assertEqual(self.other.read_bytes(),b'jv-bbbbbb world')

    def test_same_instance_atomic_replacement_original_preserved_and_isolation(self):
        other=self.other.read_bytes();plan=self.prepare()
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')
        self.restore('shutdown');self.restore('activate')
        self.assertEqual((self.data/'level.dat').read_bytes(),b'jv-aaaaaa world')
        self.assertEqual((self.tx()/'original/level.dat').read_bytes(),b'original at restore time')
        self.restore('verifying');self.restore('completed')
        self.assertEqual(self.journal()['stage'],'completed')
        self.assertEqual(self.other.read_bytes(),other)
        self.assertEqual(list((self.backups/'jv-aaaaaa').glob('minecraft-*.tar.gz')),[])
        self.assertEqual((self.backups/'minecraft-paper-2026-01-01-0000.tar.gz').read_text(),'legacy evidence')

    def test_foreign_identity_and_software_or_version_refused(self):
        path=self.backups/'jv-aaaaaa'/(self.archive+'.meta.json')
        original=json.loads(path.read_text())
        for key,value in [('id','jv-bbbbbb'),('software','vanilla'),('version','1.20.1'),('version','LATEST')]:
            m=dict(original);m[key]=value;self.private(path,json.dumps(m))
            self.restore('plan',ok=False)
        self.private(path,json.dumps(original))
        self.restore('plan',identity='jv-bbbbbb',ok=False)
        self.restore('plan',archive='minecraft-paper-2026-01-01-0000.tar.gz',ok=False)

    def test_historical_archive_names_remain_restorable(self):
        folder=self.backups/'jv-aaaaaa'
        old='instance-backup-paper-2026-10-15-1200.tar.gz'
        (folder/self.archive).rename(folder/old)
        (folder/(self.archive+'.meta.json')).unlink()
        self.item['archive']=old
        self.private(folder/(old+'.meta.json'),json.dumps(self.item))
        self.archive=old
        self.assertEqual(self.plan()['integrity'],'verified')

    def test_managed_ownership_round_trips_through_ordinary_restore(self):
        for gid in (24027, 29031):
            with self.subTest(gid=gid):
                self.env['FIXTURE_MANAGEMENT_GID']=str(gid)
                transaction='.justvoxel/plugin-transactions/ccebee6b-1102-4422-a04d-369c96868004'
                paths={'.justvoxel':0o700, '.justvoxel/plugin-transactions':0o700, transaction:0o700,
                       '.justvoxel/plugins.json':0o600, 'plugins/managed.jar':0o644, 'mods/managed.jar':0o644,
                       transaction+'/staged.jar':0o644}
                for name in ('plugins', 'mods'):
                    (self.data/name).mkdir(exist_ok=True)
                    os.chown(self.data/name,1000,1000)
                for name, mode in paths.items():
                    path=self.data/name
                    if mode == 0o700: path.mkdir(exist_ok=True)
                    else: path.write_bytes(name.encode())
                    os.chown(path,0,gid)
                    path.chmod(mode)
                known={item['archive'] for item in self.inventory()}
                self.run_helper('create')
                new=next(item for item in self.inventory() if item['archive'] not in known)
                self.archive=new['archive']
                plan=self.prepare()
                original_archive=(self.backups/'jv-aaaaaa'/self.archive).read_bytes()
                self.restore('shutdown');self.restore('activate')
                for name, mode in paths.items():
                    for location in (self.data, self.tx()/'original'):
                        info=(location/name).stat()
                        self.assertEqual((info.st_uid,info.st_gid,info.st_mode & 0o777),(0,gid,mode))
                        if mode != 0o700: self.assertEqual((location/name).read_bytes(),name.encode())
                self.restore('rollback');self.restore('rollback-verifying');self.restore('rolled-back')
                self.assertEqual(plan['integrity'],'verified')
                self.assertEqual((self.backups/'jv-aaaaaa'/self.archive).read_bytes(),original_archive)

    def test_managed_archive_ownership_and_modes_are_scoped(self):
        gid=int(self.env['FIXTURE_MANAGEMENT_GID'])
        path=self.backups/'jv-aaaaaa'/self.archive
        baseline=path.read_bytes()
        for name, uid, group, mode, code in [
            ('data/mods/managed.jar',0,gid+1,0o644,'invalid_archive_ownership'),
            ('data/mods/managed.jar',1000,gid,0o644,'invalid_archive_ownership'),
            ('data/mods/managed.jar',27123,1000,0o644,'invalid_archive_ownership'),
            ('data/config/unrelated.txt',0,gid,0o644,'invalid_archive_ownership'),
            ('data/plugins/managed.jar',0,gid,0o664,'invalid_archive_mode'),
            ('data/plugins/managed.jar',0,gid,0o646,'invalid_archive_mode'),
            ('data/plugins/managed.jar',0,0,0o4644,'invalid_archive_mode'),
            ('data/.justvoxel/plugins.json',0,gid,0o600,'invalid_archive_ownership'),
        ]:
            with self.subTest(name=name,uid=uid,group=group,mode=mode):
                with tarfile.open(fileobj=io.BytesIO(baseline),mode='r:gz') as src, tarfile.open(path,'w:gz') as dst:
                    for member in src:
                        dst.addfile(member,src.extractfile(member) if member.isfile() else None)
                    member=tarfile.TarInfo(name)
                    member.uid,member.gid,member.mode,member.size=uid,group,mode,1
                    dst.addfile(member,io.BytesIO(b'x'))
                self.replace_archive(raw=path.read_bytes())
                result=self.restore('plan',ok=False)
                self.assertEqual(json.loads(result.stdout),{'error':code})

    def test_verified_initial_leaf_archive_is_private_to_switch_clone(self):
        import shutil
        source, clone='jv-aaaaaa','jv-bbbbbb'
        for identity, software in ((source,'leaf'),(clone,'paper')):
            config=self.configs/identity/'config.json'
            record=json.loads(config.read_text())
            record['runtime'].update(software=software,version='26.3',version_policy='pinned',start_on_boot=False)
            self.private(config,json.dumps(record))
        shutil.rmtree(self.data/'world')
        (self.data/'level.dat').unlink()
        os.chown(self.data,1000,1000)
        for name,value in (('eula.txt','eula=true\n'),('.rcon-cli.env','private-runtime-secret'),
                           ('.rcon-cli.yaml','private-runtime-secret')):
            self.private(self.data/name,value)
        (self.data/'plugins/Geyser-Spigot').mkdir(parents=True)
        (self.data/'plugins/Geyser-Spigot/config.yml').write_text('test: initial configuration\n')
        jar=self.data/'plugins/Geyser-Spigot.jar'
        jar.write_bytes(b'initial executable')
        gid=int(self.env['FIXTURE_MANAGEMENT_GID'])
        os.chown(jar,0,gid)
        jar.chmod(0o644)
        def reviewed():
            return json.loads(self.restore('switch-initial-data',archive='').stdout)['fingerprint']
        proof=reviewed()
        for invalid in ('world','empty-custom-level','logs','server.jar'):
            path=self.data/invalid
            if invalid.endswith('.jar'): path.touch()
            else: path.mkdir()
            self.restore('switch-initial-data',archive='',ok=False)
            if path.is_dir(): path.rmdir()
            else: path.unlink()
        self.assertNotEqual(reviewed(),proof)
        proof=reviewed()
        self.run_helper('create',source)
        self.archive=next(item['archive'] for item in self.inventory(source) if item['software']=='leaf')
        self.assertEqual(json.loads(self.restore('plan',ok=False).stdout),{'error':'invalid_archive_world'})
        args=dict(identity=clone,clone_source=source)
        self.assertEqual(json.loads(self.restore('plan',ok=False,**args).stdout),{'error':'invalid_archive_world'})
        plan=self.plan(**args,initialization=proof)
        jar.write_bytes(b'changed executable')
        result=self.restore('prepare',plan['fingerprint'],ok=False,**args,initialization=proof)
        self.assertEqual(json.loads(result.stdout),{'error':'invalid_initialization_evidence'})
        changed=reviewed()
        result=self.restore('plan',ok=False,**args,initialization=changed)
        self.assertEqual(json.loads(result.stdout),{'error':'invalid_initialization_evidence'})
        jar.write_bytes(b'initial executable')
        proof=reviewed()
        plan=self.plan(**args,initialization=proof)
        self.restore('prepare',plan['fingerprint'],**args,initialization=proof)
        self.restore('shutdown',identity=clone);self.restore('activate',identity=clone)
        dest=self.other.parent
        self.assertEqual((dest/'plugins/Geyser-Spigot.jar').read_bytes(),jar.read_bytes())
        self.assertEqual((dest/'plugins/Geyser-Spigot.jar').stat().st_gid,gid)
        self.assertEqual((dest/'plugins/Geyser-Spigot/config.yml').read_text(),'test: initial configuration\n')
        self.assertFalse((dest/'world').exists())
        self.assertFalse((dest/'.rcon-cli.env').exists())
        self.assertFalse((dest/'.rcon-cli.yaml').exists())
        self.restore('rollback',identity=clone)
        self.restore('rollback-verifying',identity=clone);self.restore('rolled-back',identity=clone)
        self.assertTrue(self.other.exists())
        self.assertEqual((self.data/'.rcon-cli.env').read_text(),'private-runtime-secret')
        self.assertEqual((jar.stat().st_uid,jar.stat().st_gid,jar.stat().st_mode & 0o777),(0,gid,0o644))

    def test_sha_truncation_and_malformed_tar(self):
        self.assertEqual(self.plan()['integrity'],'verified')
        path=self.backups/'jv-aaaaaa'/self.archive;original=path.read_bytes()
        path.write_bytes(original[:-4]+b'xxxx')
        self.assertEqual(json.loads(self.restore('plan',ok=False).stdout),{'error':'invalid_archive_checksum'})
        for malformed in (original[:-4],gzip.compress(b'not a tar archive')):
            self.replace_archive(raw=malformed)
            self.assertEqual(json.loads(self.restore('plan',ok=False).stdout),{'error':'invalid_archive_entries'})

    def test_paths_links_duplicates_conflicts_and_special_files(self):
        for entries in [[('../escape','file',b'x')],[('/data/escape','file',b'x')],[('data/../escape','file',b'x')],[('data/link','symlink','/etc/passwd')],[('data/fifo','fifo','')],[('data/link','hardlink','data/world/level.dat')],[('data/a','file',b'a'),('data/a','file',b'b')],[('data/a/b','file',b'a'),('data/a','file',b'b')],[('data/a','file',b'a'),('data/a/b','file',b'b')]]:
            with self.subTest(entries=entries):
                self.replace_archive(entries);self.restore('plan',ok=False)
        self.assertFalse((self.root/'escape').exists())

    def test_expansion_entry_and_capacity_bounds(self):
        self.replace_archive([('data/huge','file',b'x'*8192)])
        bounded=self.restore_source.replace('MAX_BYTES = 128 * 1024**3','MAX_BYTES = 1024')
        self.restore('plan',source=bounded,ok=False)
        bounded=self.restore_source.replace('MAX_ENTRIES = 250000','MAX_ENTRIES = 1')
        self.restore('plan',source=bounded,ok=False)
        low=self.restore_source.replace('available = os.fstatvfs(parent)',"available = type('Space',(),dict(f_bavail=0,f_frsize=4096,f_favail=1000))()")
        self.restore('plan',source=low,ok=False)
        self.assertFalse((self.configs/'jv-aaaaaa/restore-journal.json').exists())

    def test_stale_plan_archive_removed_and_preparation_failure_preserves_original(self):
        plan=self.plan();meta=self.backups/'jv-aaaaaa'/(self.archive+'.meta.json')
        m=json.loads(meta.read_text());m['name']='changed';self.private(meta,json.dumps(m))
        self.restore('prepare',plan['fingerprint'],ok=False)
        self.assertFalse((self.configs/'jv-aaaaaa/restore-journal.json').exists())
        self.private(meta,json.dumps(self.item))
        plan=self.plan();(self.backups/'jv-aaaaaa'/self.archive).unlink()
        self.restore('prepare',plan['fingerprint'],ok=False)
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')

    def test_wrong_mount_and_unavailable_network_source(self):
        self.config(network=True)
        self.env['FIXTURE_MOUNT_ID']='foreign';self.restore('plan',ok=False)
        self.env['FIXTURE_MOUNT_ID']='expected'
        self.tool(self.bin/'mountpoint','#!/usr/bin/bash\nexit 1\n')
        self.restore('plan',ok=False)
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')

    def test_destination_mount_identity_using_real_prerequisite_helper(self):
        config=self.configs/'jv-aaaaaa/config.json'
        record=json.loads(config.read_text());storage=record['storage']
        storage['mount_point']=storage['root'];storage['uuid']='expected'
        self.private(config,json.dumps(record))
        common=self.helpers/'common.sh'
        common.write_text(common.read_text()+'\nrequire_root() { [[ $EUID == 0 ]]; }\n')
        source=(REPO/'mjust/libexec/instance-runtime-prerequisites').read_text()
        source=source.replace('/usr/libexec/justvoxel/mjust/',str(self.helpers)+'/')
        self.tool(self.helpers/'instance-runtime-prerequisites',source)
        self.data.parent.parent.chmod(0o700);self.data.parent.chmod(0o700)
        os.chown(self.data,1000,1000)
        self.env['FIXTURE_MOUNT_ID']='foreign';self.restore('plan',ok=False)
        self.env['FIXTURE_MOUNT_ID']='expected';self.assertEqual(self.plan()['integrity'],'verified')
        self.tool(self.bin/'mountpoint','#!/usr/bin/bash\nexit 1\n')
        self.restore('plan',ok=False)
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')

    def test_archive_and_metadata_ownership_modes_and_links(self):
        archive=self.backups/'jv-aaaaaa'/self.archive
        meta=self.backups/'jv-aaaaaa'/(self.archive+'.meta.json')
        for path in (archive,meta):
            path.chmod(0o644);self.restore('plan',ok=False);path.chmod(0o600)
        for path in (archive,meta):
            original=path.read_bytes();path.unlink()
            target=self.root/'foreign-file';target.write_bytes(original);target.chmod(0o600)
            path.symlink_to(target);self.restore('plan',ok=False);path.unlink()
            path.write_bytes(original);path.chmod(0o600)
            alias=self.root/'hardlink';os.link(path,alias)
            self.restore('plan',ok=False);alias.unlink()
        self.assertEqual(self.plan()['integrity'],'verified')

    def test_recovery_after_atomic_exchange_and_retry_without_backup_source(self):
        self.prepare();self.restore('shutdown')
        crash=self.restore_source.replace("stage(j,'replacement_activated')","raise OSError('injected interruption after atomic exchange')")
        self.restore('activate',source=crash,ok=False)
        self.assertEqual((self.data/'level.dat').read_bytes(),b'jv-aaaaaa world')
        # An interrupted restore cannot accept another prepare/plan.
        self.restore('plan',ok=False)
        (self.backups/'jv-aaaaaa'/self.archive).unlink()
        # Incomplete rollback preserves both original and failed replacement.
        broken=self.restore_source.replace("move(original_fd,original_name,parent,'data')","raise OSError('injected rollback interruption')")
        self.restore('rollback',source=broken,ok=False)
        self.assertTrue((self.tx()/'failed-replacement/level.dat').exists())
        self.restore('rollback');self.restore('rollback')
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')
        self.restore('rollback-verifying');self.restore('rolled-back')
        self.assertEqual(self.journal()['stage'],'rolled_back')
        self.assertTrue((self.tx()/'failed-replacement/level.dat').exists())

    def test_interrupted_preparation_has_recovery_even_before_transaction_mkdir(self):
        plan=self.plan()
        crash=self.restore_source.replace("os.mkdir(txname,0o700,dir_fd=parent)","raise OSError('injected early interruption')\n            os.mkdir(txname,0o700,dir_fd=parent)")
        self.restore('prepare',plan['fingerprint'],source=crash,ok=False)
        self.restore('rollback');self.restore('rollback-verifying');self.restore('rolled-back')
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')

    def test_complete_tar_end_markers_and_no_hidden_trailing_archive(self):
        self.replace_archive([('data/file','file',b'world')])
        path=self.backups/'jv-aaaaaa'/self.archive
        raw=gzip.decompress(path.read_bytes())
        with tarfile.open(fileobj=io.BytesIO(raw),mode='r:') as tf:
            members=tf.getmembers();end=members[-1].offset_data+((members[-1].size+511)//512)*512
        self.replace_archive(raw=gzip.compress(raw[:end]));self.restore('plan',ok=False)
        self.replace_archive(raw=gzip.compress(raw+b'hidden trailing data'));self.restore('plan',ok=False)

    def test_world_version_and_corrupt_world_metadata(self):
        path=self.backups/'jv-aaaaaa'/self.archive
        original=path.read_bytes()
        for value,code in ((world_metadata('1.22.0'),'incompatible_software_version'),(gzip.compress(b'invalid NBT'),'invalid_archive_world')):
            with tarfile.open(path,'w:gz') as tf:
                root=tarfile.TarInfo('data');root.type=tarfile.DIRTYPE;root.mode=0o750;tf.addfile(root)
                member=tarfile.TarInfo('data/world/level.dat');member.mode=0o640;member.size=len(value)
                tf.addfile(member,io.BytesIO(value))
            self.replace_archive(raw=path.read_bytes())
            self.assertEqual(json.loads(self.restore('plan',ok=False).stdout),{'error':code})
        self.replace_archive(raw=original)
        self.assertEqual(self.plan()['integrity'],'verified')

    def test_retention_preserves_interrupted_restore_archive(self):
        meta=self.backups/'jv-aaaaaa'/(self.archive+'.meta.json')
        m=json.loads(meta.read_text());m['created']='2020-01-01T00:00:00Z';self.private(meta,json.dumps(m))
        self.prepare()
        for _ in range(3):self.run_helper('create')
        self.run_helper('retain') # Fixture startup validator is mocked for this defense-in-depth check.
        self.assertTrue((self.backups/'jv-aaaaaa'/self.archive).exists())
        self.assertTrue((self.backups/'jv-aaaaaa'/(self.archive+'.meta.json')).exists())
        self.restore('rollback');self.restore('rollback-verifying');self.restore('rolled-back')
        self.run_helper('retain')
        self.assertFalse((self.backups/'jv-aaaaaa'/self.archive).exists())

    def test_rollback_before_exchange_and_after_original_preservation(self):
        self.prepare();self.restore('shutdown')
        crash=self.restore_source.replace("stage(j,'preserving_original')","raise OSError('before exchange')")
        self.restore('activate',source=crash,ok=False)
        self.restore('rollback')
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')
        self.restore('rollback-verifying');self.restore('rolled-back')
        self.prepare();self.restore('shutdown');self.restore('activate');self.restore('rollback')
        self.assertEqual((self.data/'level.dat').read_bytes(),b'original at restore time')
        self.assertTrue((self.tx()/'failed-replacement/level.dat').exists())

unittest.main()
PY
