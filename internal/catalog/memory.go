package catalog

import (
	"strings"

	"food-ordering-api/internal/models"
)

func imgURLs(base, slug string) models.ProductImage {
	b := strings.TrimRight(strings.TrimSpace(base), "/") + "/"
	return models.ProductImage{
		Thumbnail: b + "image-" + slug + "-thumbnail.jpg",
		Mobile:    b + "image-" + slug + "-mobile.jpg",
		Tablet:    b + "image-" + slug + "-tablet.jpg",
		Desktop:   b + "image-" + slug + "-desktop.jpg",
	}
}

// MemoryCatalog is an in-memory implementation of Catalog.
// For production this would be replaced with a repository backed by PostgreSQL,
// read replicas, or a product service — the interface stays the same.
type MemoryCatalog struct {
	byID map[string]models.Product
	list []models.Product
}

// Catalog exposes read-only product data to HTTP handlers.
type Catalog interface {
	List() []models.Product
	ByID(id string) (models.Product, bool)
}

// NewMemory builds the static catalogue used by the challenge demo.
// imageBase is typically IMAGE_BASE_URL (e.g. https://host/.../images/).
func NewMemory(imageBase string) *MemoryCatalog {
	list := []models.Product{
		{ID: "1", Image: imgURLs(imageBase, "waffle"), Name: "Waffle with Berries", Category: "Waffle", Price: 6.5},
		{ID: "2", Image: imgURLs(imageBase, "creme-brulee"), Name: "Vanilla Bean Crème Brûlée", Category: "Crème Brûlée", Price: 7},
		{ID: "3", Image: imgURLs(imageBase, "macaron"), Name: "Macaron Mix of Five", Category: "Macaron", Price: 8},
		{ID: "4", Image: imgURLs(imageBase, "tiramisu"), Name: "Classic Tiramisu", Category: "Tiramisu", Price: 5.5},
		{ID: "5", Image: imgURLs(imageBase, "baklava"), Name: "Pistachio Baklava", Category: "Baklava", Price: 4},
		{ID: "6", Image: imgURLs(imageBase, "meringue"), Name: "Lemon Meringue Pie", Category: "Pie", Price: 5},
		{ID: "7", Image: imgURLs(imageBase, "cake"), Name: "Red Velvet Cake", Category: "Cake", Price: 4.5},
		{ID: "8", Image: imgURLs(imageBase, "brownie"), Name: "Salted Caramel Brownie", Category: "Brownie", Price: 4.5},
		{ID: "9", Image: imgURLs(imageBase, "panna-cotta"), Name: "Vanilla Panna Cotta", Category: "Panna Cotta", Price: 6.5},
	}
	byID := make(map[string]models.Product, len(list))
	for _, p := range list {
		byID[p.ID] = p
	}
	return &MemoryCatalog{byID: byID, list: list}
}

// List returns products in stable order.
func (m *MemoryCatalog) List() []models.Product {
	return m.list
}

// ByID returns a product by string id.
func (m *MemoryCatalog) ByID(id string) (models.Product, bool) {
	p, ok := m.byID[id]
	return p, ok
}
