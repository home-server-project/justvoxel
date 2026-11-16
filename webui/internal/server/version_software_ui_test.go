package server

import (
	"fmt"
	"strings"
	"testing"
)

func TestMigrationSwitchCardsAndReviewedActions(t *testing.T) {
	script, err := englishPresentationAsset("static/server-migration-switch.js")
	if err != nil {
		t.Fatal(err)
	}
	program := `
const assert = require("node:assert/strict");
const vm = require("node:vm");
const source = ` + fmt.Sprintf("%q", string(script)) + `;
class Element {
 constructor(tag="div") { this.tagName=tag; this.dataset={}; this.children=[]; this.events={}; this.disabled=false; this.hidden=false; this.checked=false; this.isConnected=true; this.attributes={}; this.ownText=""; this.classes=new Set(); this.classList={toggle:(key,value)=>value?this.classes.add(key):this.classes.delete(key)}; }
 set textContent(value) {this.ownText=String(value);this.children=[];}
 get textContent() {return this.ownText+this.children.map(child=>child.textContent).join(" ");}
 append(...nodes) {this.children.push(...nodes);}
 appendChild(node) {this.append(node);return node;}
 replaceChildren(...nodes) {this.children=[];this.ownText="";this.append(...nodes);}
 setAttribute(key,value) {this.attributes[key]=value;}
 addEventListener(key,fn) {
  const listeners=this.eventListeners ||= {};
  (listeners[key] ||= []).push(fn);
  this.events[key]=(...args)=>Promise.all(listeners[key].map(listener=>listener(...args)));
 }
 all() {return this.children.flatMap(child=>[child,...child.all()]);}
 querySelectorAll(selector) {if(selector==="[data-switch-target]") return this.cards;return this.all().filter(node=>node.tagName===selector);}
 querySelector(selector) {return this.selectors?.[selector];}
 matches(selector) {return selector==="[data-migration-switch]";}
}
const identities=["paper","purpur","spigot","pufferfish","leaf","fabric","neoforge","forge","quilt","vanilla","bedrock"];
const flush=()=>new Promise(resolve=>setImmediate(resolve));
async function open(fail=false) {
 const root=new Element();root.dataset.serverId="jv-family1";
 const state=new Element(),review=new Element(),progress=new Element(),csrf=new Element("input");csrf.value="csrf-token";
 root.selectors={"[data-switch-state]":state,"[data-switch-review]":review,"[data-switch-operation]":progress,"[data-switch-csrf]":csrf};
 root.cards=identities.map(id=>{const node=new Element("button");node.dataset.switchTarget=id;node.disabled=true;node.selectors={"[data-switch-sticker]":new Element("span")};return node;});
 const calls=[];let applied=null;
 const fetch=async(raw,options)=>{
  const url=new URL(raw,"http://example");calls.push({url,options});assert.equal(url.searchParams.get("server"),"jv-family1");
  let result;
  if(url.pathname.endsWith("/operation")) result=applied?{operation:{operation_id:"switch1",state:fail?"needs_attention":"succeeded",status:fail?"Switch did not complete.":"Switch completed."}}:{};
  else if(url.pathname.endsWith("/plan")) {const target=url.searchParams.get("target");result={instance_id:"jv-family1",current:"paper",target,version:"1.21.4",convert:target==="purpur",clone:["purpur","pufferfish"].includes(target),plan_fingerprint:"reviewed-"+target,content:target==="pufferfish"?[{name:"FamilyTools",filename:"family.jar",compatibility:"unknown",reason:"Unverified provenance"}]:[]};}
  else if(url.pathname.endsWith("/apply")) {applied=options.body;result={operation:{operation_id:"switch1"}};}
  else throw Error("Unexpected Switch request: "+url);
  return {ok:true,json:async()=>result};
 };
 const window={};vm.runInNewContext(source,{window,document:{createElement:tag=>new Element(tag)},justVoxelSelectionGeneration:{migration:0},fetch,URLSearchParams,setTimeout:()=>{}});
 window.JustVoxelServerMigrationSwitch.init(root);await flush();await flush();
 return {root,state,review,progress,calls};
}
(async()=>{
 const page=await open();const cards=page.root.cards;
 assert.equal(cards.length,11);
 assert.equal(cards[0].querySelector("[data-switch-sticker]").textContent,"Current");assert.equal(cards[0].disabled,true);
 assert.equal(cards[1].querySelector("[data-switch-sticker]").textContent,"Available");assert.equal(cards[1].disabled,false);
 for(const id of ["spigot","fabric","neoforge","forge","quilt","vanilla","bedrock"]) {const card=cards.find(node=>node.dataset.switchTarget===id);assert.equal(card.disabled,true);assert.equal(card.querySelector("[data-switch-sticker]").textContent,"Not Supported");}
 cards[3].events.click();await flush();assert.match(page.review.textContent,/FamilyTools.*Unknown \/ unverified/);
 const buttons=page.review.all().filter(node=>node.tagName==="button");const consent=page.review.all().find(node=>node.tagName==="input");consent.checked=true;consent.events.change();
 assert.equal(buttons.find(node=>node.textContent==="Convert existing server").disabled,true);assert.equal(buttons.find(node=>node.textContent==="Clone and convert").disabled,false);
 cards[1].events.click();await flush();const confirm=page.review.all().find(node=>node.tagName==="input");
 const convert=page.review.all().find(node=>node.tagName==="button"&&node.textContent==="Convert existing server");
 const clone=page.review.all().find(node=>node.tagName==="button"&&node.textContent==="Clone and convert");
 assert.equal(confirm.checked,false);assert.equal(convert.disabled,true);assert.equal(clone.disabled,true);
 await convert.events.click();assert.equal(page.calls.filter(call=>call.options.method==="POST").length,0);
 confirm.checked=true;await confirm.events.change();assert.equal(convert.disabled,false);assert.equal(clone.disabled,false);
 await convert.events.click();await convert.events.click();await clone.events.click();await flush();
 const posts=page.calls.filter(call=>call.options.method==="POST");assert.equal(posts.length,1);assert.equal(posts[0].url.pathname,"/api/migration/switch/apply");assert.equal(posts[0].options.body.get("target"),"purpur");assert.equal(posts[0].options.body.get("plan_fingerprint"),"reviewed-purpur");assert.equal(posts[0].options.body.get("action"),"convert");assert.equal(posts[0].options.body.get("confirm"),"yes");assert.equal(posts[0].options.body.has("version"),false);
 const failure=await open(true);failure.root.cards[3].events.click();await flush();const c=failure.review.all().find(node=>node.tagName==="input");c.checked=true;c.events.change();await failure.review.all().find(node=>node.textContent==="Clone and convert").events.click();await flush();
 const clonePosts=failure.calls.filter(call=>call.options.method==="POST");assert.equal(clonePosts.length,1);assert.equal(clonePosts[0].url.pathname,"/api/migration/switch/apply");assert.equal(clonePosts[0].options.body.get("action"),"clone");assert.equal(clonePosts[0].options.body.get("target"),"pufferfish");assert.equal(clonePosts[0].options.body.get("plan_fingerprint"),"reviewed-pufferfish");assert.equal(clonePosts[0].options.body.get("confirm"),"yes");assert.equal(clonePosts[0].options.body.has("version"),false);
 assert.match(failure.progress.textContent,/Switch did not complete/);assert.doesNotMatch(failure.progress.textContent,/Clone created and verified/);assert.match(failure.progress.textContent,/Recover Switch/);
})().catch(error=>{console.error(error);process.exitCode=1;});
`
	cmd := nodeTestCommand()
	cmd.Stdin = strings.NewReader(program)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Migration Switch behavior: %v\n%s", err, output)
	}
}

func TestSetupSimpleSoftwareCardWording(t *testing.T) {
	data, err := englishPresentationAsset("templates/setup_wizard.html")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, old := range []string{"Recommended · plugins · Bedrock cross-play enabled by default when supported.", "Use recommended setup"} {
		if strings.Contains(text, old) {
			t.Fatalf("old First Setup presentation remains: %s", old)
		}
	}
	for _, want := range []string{`{{template "setup-simple-card" .}}`, `{{range .SimpleSoftware}}`, `src="/static/server-software/{{.ID}}.svg"`, `class="server-software-copy"`, `<strong>{{.Name}}</strong><small>{{.Description}}</small>`} {
		if !strings.Contains(text, want) {
			t.Fatalf("shared software card missing %s", want)
		}
	}
	if simpleSoftware("plugins")[0].Description != "Plugins · Cross-play" {
		t.Fatal("Paper Simple description changed")
	}
}
