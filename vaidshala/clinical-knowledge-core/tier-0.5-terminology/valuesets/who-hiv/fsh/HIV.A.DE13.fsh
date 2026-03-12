ValueSet: HIV.A.DE13
Title: "Country of birth ValueSet"
Description: "Value set of country where the client was born"
* ^meta.profile[+] = "http://hl7.org/fhir/uv/crmi/StructureDefinition/crmi-shareablevalueset"
* ^meta.profile[+] = "http://hl7.org/fhir/uv/crmi/StructureDefinition/crmi-publishablevalueset"
* ^meta.profile[+] = "http://hl7.org/fhir/uv/crmi/StructureDefinition/crmi-computablevalueset"
* ^status = #active
* ^experimental = true
* ^name = "HIVADE13"
* ^url = "http://smart.who.int/hiv/ValueSet/HIV.A.DE13"
* ^extension[+].url = "http://hl7.org/fhir/StructureDefinition/valueset-rules-text"
* ^extension[=].valueMarkdown = "This should be a context-specific list of countries where a patient might be born"
