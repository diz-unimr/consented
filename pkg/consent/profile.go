package consent

import (
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
)

const (
	MiiDomainName          = "MII"
	MiiSignerIdPrefix      = "https://ths-greifswald.de/fhir/gics/identifiers/"
	MiiTargetPolicySystem  = "https://ths-greifswald.de/fhir/CodeSystem/gics/Policy"
	MiiTargetPolicyCode    = "MDAT_erheben"
	MiiTargetPolicyVersion = "1.1"
)

type Profile struct {
	Domain         string
	PolicyCoding   *fhir.Coding
	PolicyVersion  *string
	PersonIdSystem *string
}

func GetSupportedProfiles(signerId string) map[string]*Profile {

	return map[string]*Profile{
		MiiDomainName: {
			Domain: MiiDomainName,
			PolicyCoding: &fhir.Coding{
				System: Of(MiiTargetPolicySystem),
				Code:   Of(MiiTargetPolicyCode),
			},
			PolicyVersion:  Of(MiiTargetPolicyVersion),
			PersonIdSystem: Of(MiiSignerIdPrefix + signerId),
		},
	}
}

func Of[E any](e E) *E {
	return &e
}
