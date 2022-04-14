package model

type Structure struct {
	Name          string
	EmbedNames    []string      // used for pkg-local embeds
	ImportEmbeds  []ImportEmbed // used for embeds from imported pkg
	FieldTagNames map[string]FieldTagNames
}

func (s *Structure) AddFieldTagName(tag, fieldName, tagName string) {
	if s.FieldTagNames == nil {
		s.FieldTagNames = make(map[string]FieldTagNames)
	}
	if _, exists := s.FieldTagNames[tag]; !exists {
		s.FieldTagNames[tag] = FieldTagNames{}
	}
	s.FieldTagNames[tag] = append(s.FieldTagNames[tag], FieldTagName{FieldName: fieldName, TagName: tagName})
}

type Structures []*Structure

func (s Structures) ByName(name string) *Structure {
	for i := range s {
		if s[i].Name == name {
			return s[i]
		}
	}
	return nil
}

func (s Structures) HavingTags(tags []string) Structures {
	strucs := Structures{}
nextStruct:
	for idx := range s {
		for _, t := range tags {
			if _, exists := s[idx].FieldTagNames[t]; exists {
				strucs = append(strucs, s[idx])
				continue nextStruct
			}
		}
	}

	return strucs
}

// returns all fully-qualified pkg names of embedded imports
func (s Structures) EmbeddedImportPkgNames() []string {
	imps := make(map[string]bool)
	for idx := range s {
		for _, ie := range s[idx].ImportEmbeds {
			if _, exists := imps[ie.FullyQualifiedPkgName]; !exists && ie.FullyQualifiedPkgName != "" {
				imps[ie.FullyQualifiedPkgName] = true
			}
		}
	}
	ret := make([]string, 0)
	for imp := range imps {
		ret = append(ret, imp)
	}
	return ret
}

type FieldTagName struct {
	FieldName string
	TagName   string
}

func (ftn FieldTagName) IsSkipped() bool {
	switch ftn.TagName {
	case "-", "ignore":
		return true
	default:
		return false
	}
}

type FieldTagNames []FieldTagName

func (ftns FieldTagNames) AllSkipped() bool {
	for _, ftn := range ftns {
		if !ftn.IsSkipped() {
			return false
		}
	}
	return true
}

func (ftns FieldTagNames) TagNames() []string {
	t := []string{}
	for _, tn := range ftns {
		if !tn.IsSkipped() {
			t = append(t, tn.TagName)
		}
	}
	return t
}
