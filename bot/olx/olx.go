package olx

type OLXAd struct {
	Id       int
	Title    string
	Image    string
	Price    int
	Location string
}

var Categories = []string{
	"Eletrônicos e Celulares",
	"Para a Sua Casa",
	"Eletro",
	"Móveis",
	"Esportes e Lazer",
	"Música e Hobbies",
	"Agro e Indústria",
	"Roupas",
	"Artigos Infantis",
	"Animais de Estimação",
	"Câmeras e Drones",
	"Games",
	"Escritório",
	"Carros, vans e utilitários",
}

const OLX_MAX_PRICE = 99_999_999
