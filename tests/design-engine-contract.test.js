import test from 'node:test';
import assert from 'node:assert/strict';
import {DESIGN_REGISTRY,DEFAULT_PREFERENCES,ACCENTS,normalizePreferences,resolveDesign,contrast,createDesignProvider,STORAGE_KEY} from '../web/design-engine.js';

test('Adaptive engine registers eight distinct complete design token sets',()=>{
 assert.equal(DESIGN_REGISTRY.length,8);assert.equal(new Set(DESIGN_REGISTRY.map(item=>item.id)).size,8);
 const shapes=new Set();for(const design of DESIGN_REGISTRY){const {tokens}=resolveDesign({style:design.id,mode:'dark',accent:'blue'});for(const key of ['bg','panel','text','muted','quiet','blue','radius','shadow','design-space','design-blur','design-font','design-focus','design-duration','design-easing','design-hover','design-selected','design-disabled-opacity'])assert.ok(tokens[key],`${design.id}: ${key}`);shapes.add(JSON.stringify(design.tokens));}assert.equal(shapes.size,8);
});

test('All new design palettes retain AA normal-text and accent contrast in both modes',()=>{
 for(const design of DESIGN_REGISTRY)for(const mode of ['dark','light'])for(const accent of Object.keys(ACCENTS)){const {tokens}=resolveDesign({style:design.id,mode,accent});for(const surface of ['bg','panel','panel2','panel3'])for(const text of ['text','muted','quiet','blue','cyan','violet'])assert.ok(contrast(tokens[text],tokens[surface])>=4.5,`${design.id}/${mode}/${accent} ${text} on ${surface}: ${contrast(tokens[text],tokens[surface])}`);assert.ok(contrast(tokens['design-on-accent'],tokens['design-accent'])>=4.5);}
});

test('Preferences allow only known non-sensitive style, mode and accent values',()=>{
 assert.deepEqual(normalizePreferences({style:'javascript:bad',mode:'remote',accent:'url(secret)',token:'sensitive'}),DEFAULT_PREFERENCES);
 assert.equal(Object.keys(normalizePreferences({style:'minimalism',mode:'light',accent:'teal',csrf:'private'})).length,4);
 assert.equal(resolveDesign(DEFAULT_PREFERENCES).tokens.bg,'#060816');assert.equal(resolveDesign(DEFAULT_PREFERENCES).tokens.blue,'#53a5ff');
});

test('Provider previews without persistence and supports apply, revert, restore and corrupt storage fallback',()=>{
 const saved=new Map(),properties=new Map(),root={dataset:{},style:{setProperty:(key,value)=>properties.set(key,value)},dispatchEvent(){}};
 const storage={getItem:key=>saved.get(key)||null,setItem:(key,value)=>saved.set(key,value)};const provider=createDesignProvider({root,storage});
 provider.preview({style:'glassmorphism',mode:'light',accent:'teal'});assert.equal(root.dataset.design,'glassmorphism');assert.equal(saved.has(STORAGE_KEY),false);provider.cancel();assert.equal(root.dataset.design,'original');
 provider.apply({style:'minimalism',mode:'light',accent:'violet'});const persisted=JSON.parse(saved.get(STORAGE_KEY));assert.deepEqual(persisted,{version:1,style:'minimalism',mode:'light',accent:'violet'});provider.apply({style:'retro-y2k',mode:'dark',accent:'blue'});provider.revert();assert.equal(provider.preferences.style,'minimalism');provider.restore();assert.deepEqual(provider.preferences,DEFAULT_PREFERENCES);
 saved.set(STORAGE_KEY,'not json');const recovered=createDesignProvider({root,storage});assert.equal(recovered.preferences.style,'original');assert.equal(recovered.storageAvailable,false);
 const blocked=createDesignProvider({root,storage:{getItem(){throw Error('denied');},setItem(){throw Error('denied');}}});blocked.apply({style:'bento-grid',mode:'dark',accent:'rose'});assert.equal(blocked.preferences.style,'bento-grid');assert.equal(blocked.storageAvailable,false);
});
