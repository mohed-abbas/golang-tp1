package model

// Stats résume l'état de la flotte et des vols.
type Stats struct {
	NombreAvions  int            `json:"nombre_avions"`
	NombreVols    int            `json:"nombre_vols"`
	VolsParStatut map[Statut]int `json:"vols_par_statut"`
}
