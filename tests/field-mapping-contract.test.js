import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';

const app=fs.readFileSync(new URL('../web/app.js',import.meta.url),'utf8');
const helpers=app.slice(app.indexOf('  const QRW_RESERVED='),app.indexOf('  function setupQRWStepPanels('));
const extraction=app.slice(app.indexOf('  function qrwSlice('),app.indexOf('  async function qrwValidateCandidate('));
function engine(){return vm.runInNewContext(`${helpers}\n${extraction}\n({field:qrwField,suggest:qrwSuggestedFields,merge:qrwMergeControllerSuggestions,extract:qrwExtract,tokens:qrwTokens,pairs:qrwPairs,plan:fields=>{state.qrwFields=fields;return qrwPlan();}})`,{TextEncoder,TextDecoder,state:{qrwFields:[]}});}

test('Key value EOL round trips as a line-bounded native dynamic plan while explicit multiline remains distinct',()=>{
 const e=engine(),field=e.field({name:'tail_fields',method:'kv_eol',source:'9',unit:'token',type:'object',policy:'required',output_policy:'flatten',assignment_separators:['=',':'],pair_delimiters:':;| \t\r\n',require_complete:true});
 const native=JSON.parse(JSON.stringify(e.plan([field]))).fields[0];assert.equal(native.method,'kv_all');assert.equal(native.end_of_line,true);assert.equal(native.require_complete,true);assert.deepEqual(native.assignment_separators,['=',':']);assert.equal(native.source,'9');assert.equal(native.output_policy,'flatten');assert.equal(e.field(native).method,'kv_eol');
 const explicit=e.plan([e.field({...native,end_of_line:false})]).fields[0];assert.equal(explicit.method,'kv_all');assert.equal(explicit.end_of_line,false);
});

test('Controller suggestions retain operator labels, ordered transformations and extra manual fields',()=>{
 const e=engine(),existing=[e.field({name:'my_service',label:'My service',method:'position',source:'2',operations:[{op:'prefix',value:'prod-'}]}),e.field({name:'audit_tag',method:'constant',default_value:'manual'})];
 const fields=e.merge([{name:'service',method:'position',source:'2',type:'string'},{name:'host',method:'position',source:'1',type:'string'}],existing);
 assert.equal(fields.length,3);assert.equal(fields[0].name,'my_service');assert.equal(fields[0].label,'My service');assert.equal(fields[0].operations[0].value,'prod-');assert.ok(fields.some(field=>field.name==='audit_tag'));
});

test('NFI suggests compound labels and every colon-separated key without a raw-line special case',()=>{
  const e=engine(),raw='20261201 102030123+0530 mailnode smtp 42 0 140026854221568 Note;DeliveryEvent status=failed:reason=notneeded:server=cloud';
  const fields=e.suggest(raw);assert.equal(fields.length,12);assert.equal(fields[7].name,'loglevel');assert.equal(fields[8].name,'event');
  const out={};for(const field of fields){if(field.method==='delimiter')field.confirmed=true;const result=e.extract(field,raw,out);assert.equal(result.error,undefined,result.error);out[field.name]=result.value;}
  assert.equal(out.loglevel,'Note');assert.equal(out.event,'DeliveryEvent');assert.equal(out.status,'failed');assert.equal(out.reason,'notneeded');assert.equal(out.server,'cloud');assert.equal(out.thread_id,'140026854221568');
  assert.ok(!app.includes("const legacy=text==="));assert.ok(!app.includes("const smtp=text==="));
});

test('NFI key boundaries preserve URLs, quoted colons, escapes and multiword values',()=>{
  const e=engine(),raw='status="not: needed":url=https://example.test/a?x=y:server=cloud reason=two words address=2001:db8::1';
  const field=source=>e.field({name:source,method:'key_value',source,policy:'required'});
  assert.equal(e.extract(field('status'),raw).value,'not: needed');assert.equal(e.extract(field('url'),raw).value,'https://example.test/a?x=y');assert.equal(e.extract(field('reason'),raw).value,'two words');assert.equal(e.extract(field('address'),raw).value,'2001:db8::1');
  assert.equal(e.extract(field('empty'),'empty=').value,'');assert.match(e.extract(field('status'),'status=first:status=second').error,/Duplicate/);assert.match(e.extract(field('status'),'status="unfinished').error,/Unterminated/);
});

test('NFI ordered split, replacement, case conversion and prior-field merge are deterministic',()=>{
  const e=engine(),raw='prefix;Keep|unused';
  const field=e.field({name:'action',method:'constant',default_value:raw,policy:'required',operations:[{op:'split',delimiter:';',part:2},{op:'split',delimiter:'|',part:1},{op:'lowercase'},{op:'replace',find:'keep',replace:'accept'},{op:'prefix',value:'policy:'}]});
  assert.equal(e.extract(field,raw).value,'policy:accept');
  assert.equal(e.extract(e.field({name:'combined',method:'derived',source:'a,b',join_separator:'/',policy:'derived'}),raw,{a:'one',b:'two'}).value,'one/two');
  assert.match(e.extract(e.field({name:'combined',method:'derived',source:'future',policy:'derived'}),raw,{}).error,/earlier/);
});

test('NFI Unicode character and byte spans remain distinct and byte cuts fail visibly',()=>{
  const e=engine(),raw='α beta';const tokens=e.tokens(raw);assert.equal(tokens[1].charStart,3);assert.equal(tokens[1].byteStart,4);
  assert.equal(e.extract(e.field({name:'symbol',method:'character',source:'1',source_end:'1',policy:'required'}),raw).value,'α');
  assert.equal(e.extract(e.field({name:'symbol',method:'byte',source:'1',source_end:'2',policy:'required'}),raw).value,'α');
  assert.match(e.extract(e.field({name:'symbol',method:'byte',source:'1',source_end:'1',policy:'required'}),raw).error,/UTF-8/);
});

test('NFI JSON/CSV suggestions and nested key/value operations retain selectable source evidence',()=>{
  const e=engine();const json=e.suggest('{"request":{"tags":["a","b"]},"ok":true}');assert.equal(json[0].source,'$.request.tags[0]');assert.equal(json.at(-1).type,'boolean');
  const csv=e.suggest('"west,region",007,"two words"');assert.equal(csv.length,3);for(const field of csv)field.confirmed=true;assert.equal(e.extract(csv[0],'"west,region",007,"two words"').value,'west,region');assert.equal(e.extract(csv[1],'"west,region",007,"two words"').value,'007');
  const nested=e.field({name:'action',policy:'required',method:'key_value',source:'msg',operations:[{op:'strip_prefix',value:'MISREPORT='},{op:'key_value',value:'action',delimiter:';',separator:'|'}]});assert.equal(e.extract(nested,'msg=MISREPORT=eventType;summary|action;keep|geo;US').value,'keep');
});

test('NFI refuses unsafe integer previews and executable regex work in the browser',()=>{
  const e=engine();assert.match(e.extract(e.field({name:'id',method:'constant',default_value:'9007199254740993',policy:'required',type:'integer'}),'x').error,/exact browser range/);
  assert.match(e.extract(e.field({name:'regex',method:'regex',source:'(a+)+$',policy:'required'}),'aaaa').evidence,/native RE2 preview/);
  assert.match(e.extract(e.field({name:'ops',method:'constant',default_value:'x',policy:'required',operations:Array.from({length:17},()=>({op:'trim'}))}),'x').error,/16/);
});

test('NFI preserves quote policy, array selection and empty/default/optional semantics',()=>{
  const e=engine();const selected=e.field({name:'part',method:'constant',default_value:'a|b|c',policy:'required',operations:[{op:'split',delimiter:'|',part:0},{op:'select',part:2}]});assert.equal(e.extract(selected,'').value,'b');
  assert.equal(e.extract(e.field({name:'unquoted',method:'delimiter',source:'raw',delimiter:',',part:'1',confirmed:true,quote_aware:false,policy:'required'}),'"west,region",007').value,'"west');
  assert.equal(e.extract(e.field({name:'empty',method:'key_value',source:'missing',policy:'default',default_value:''}),'other=x').value,'');
  assert.equal(e.extract(e.field({name:'missing',method:'character',source:'99',policy:'optional'}),'short').absent,true);
  assert.match(e.extract(e.field({name:'flag',method:'constant',default_value:'TRUE',type:'boolean',policy:'required'}),'').error,/boolean/);
  const suggestions=e.suggest('raw=x source=y parser_id=z');assert.deepEqual(Array.from(suggestions,field=>field.name),['raw_value','source_value','parser_id_value']);
});


test('NFI applies absence policies to missing nested keys without masking malformed values',()=>{
  const e=engine(),raw='msg=MISREPORT=action;keep|geo;US';
  const base={name:'reason',method:'key_value',source:'msg',operations:[{op:'strip_prefix',value:'MISREPORT='},{op:'key_value',value:'reason',delimiter:';',separator:'|'},{op:'prefix',value:'reason:'}]};
  assert.equal(e.extract(e.field({...base,policy:'optional'}),raw).absent,true);
  assert.equal(e.extract(e.field({...base,policy:'nullable'}),raw).value,null);
  assert.equal(e.extract(e.field({...base,policy:'default',default_value:'unavailable'}),raw).value,'unavailable');
  assert.match(e.extract(e.field({...base,policy:'required'}),raw).error,/Required nested/);
  assert.match(e.extract(e.field({...base,policy:'optional',null_policy:'reject'}),raw).error,/rejects/);
  assert.match(e.extract(e.field({...base,policy:'optional'}),'msg="unfinished').error,/Unterminated/);
  const malformed=e.field({...base,policy:'optional',type:'integer',operations:base.operations.slice(0,2)});
  assert.match(e.extract(malformed,'msg=MISREPORT=reason;invalid').error,/not an integer/);
});


test('FieldSense preserves executable advanced parameters and previews raw, word and earlier-field sources',()=>{
  const e=engine(),raw='node   smtp 007';
  assert.equal(e.extract(e.field({name:'service',method:'word',source:'2',policy:'required'}),raw).value,'smtp');
  assert.equal(e.extract(e.field({name:'full',method:'raw',policy:'required'}),raw).value,raw);
  assert.equal(e.extract(e.field({name:'copy',method:'field',source:'service',policy:'required'}),raw,{service:'smtp'}).value,'smtp');
  const field=e.field({name:'summary',method:'kv_all',type:'object',policy:'optional',assignment_separators:['=',';'],key_aliases:{src:'source_ip'},key_include:['src'],output_policy:'intermediate',empty_policy:'missing',description:'Stable custom description',example:'src=127.0.0.1',operations:[{op:'conditional_merge',fields:['a','b'],condition_field:'status',condition_operator:'equals',condition_value:'ok',else_value:'unknown',remove_sources:true}]});
  const plan=e.plan([field]);assert.equal(plan.fields[0].output_policy,'intermediate');assert.equal(plan.fields[0].empty_policy,'missing');assert.equal(plan.fields[0].key_aliases.src,'source_ip');assert.equal(plan.fields[0].operations[0].condition_operator,'equals');assert.equal(plan.fields[0].description,'Stable custom description');assert.equal(Object.hasOwn(plan.fields[0],'id'),false);
  const preview=e.extract(field,raw);
  assert.equal(preview.pending,true);
  assert.equal(preview.error,undefined);
  assert.match(preview.evidence,/shared KV engine/);
});
