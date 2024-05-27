package artifact

//func parseArgsGet(cfg *config.ArtifactDownload) (*models.JobArtifact, error) {
//
//	models.JobArtifact{
//		LastModified: "",
//		Path:         "",
//		Size:         0,
//	}
//}
//
//func unstructuredToJob(un any) (*models.JobsJob, error) {
//	raw, ok := un.(map[string]any)
//	if !ok {
//		return nil, errors.New("missing spec field")
//	}
//	spec := raw["spec"]
//
//	d, err := json.Marshal(spec)
//	if err != nil {
//		return nil, err
//	}
//	rg := &models.JobsJob{}
//	if err := jsonexact.Unmarshal(d, &rg.Spec); err != nil {
//		return nil, fmt.Errorf("couldn't parse YAML job definition - %s",
//			strings.TrimPrefix(err.Error(), "json: "))
//	}
//	return rg, nil
//}
