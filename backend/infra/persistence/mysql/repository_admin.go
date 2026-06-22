package mysql

import (
	"context"
	"encoding/json"

	domain "store-mind/domain/customerqa"

	"gorm.io/gorm"
)

// AdminRepository 实现 domain.AdminRepository，提供管理后台 CRUD 操作。
type AdminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// ---------- Store ----------

func (r *AdminRepository) CreateStore(ctx context.Context, store *domain.Store) (*domain.Store, error) {
	m := StoreModel{
		Name:    store.Name,
		Address: store.Address,
		Status:  store.Status,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	store.ID = m.ID
	store.CreatedAt = m.CreatedAt
	store.UpdatedAt = m.UpdatedAt
	return store, nil
}

func (r *AdminRepository) UpdateStore(ctx context.Context, store *domain.Store) (*domain.Store, error) {
	m := StoreModel{
		ID:      store.ID,
		Name:    store.Name,
		Address: store.Address,
		Status:  store.Status,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	store.UpdatedAt = m.UpdatedAt
	return store, nil
}

func (r *AdminRepository) DeleteStore(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&StoreModel{}, id).Error
}

// ---------- Zone ----------

func (r *AdminRepository) CreateZone(ctx context.Context, zone *domain.Zone) (*domain.Zone, error) {
	m := ZoneModel{
		StoreID:     zone.StoreID,
		Code:        zone.Code,
		Name:        zone.Name,
		Description: zone.Description,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	zone.ID = m.ID
	return zone, nil
}

func (r *AdminRepository) UpdateZone(ctx context.Context, zone *domain.Zone) (*domain.Zone, error) {
	m := ZoneModel{
		ID:          zone.ID,
		StoreID:     zone.StoreID,
		Code:        zone.Code,
		Name:        zone.Name,
		Description: zone.Description,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return zone, nil
}

func (r *AdminRepository) DeleteZone(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ZoneModel{}, id).Error
}

// ---------- Shelf ----------

func (r *AdminRepository) CreateShelf(ctx context.Context, shelf *domain.Shelf) (*domain.Shelf, error) {
	m := ShelfModel{
		StoreID:     shelf.StoreID,
		ZoneID:      shelf.ZoneID,
		Code:        shelf.Code,
		Name:        shelf.Name,
		Description: shelf.Description,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	shelf.ID = m.ID
	return shelf, nil
}

func (r *AdminRepository) UpdateShelf(ctx context.Context, shelf *domain.Shelf) (*domain.Shelf, error) {
	m := ShelfModel{
		ID:          shelf.ID,
		StoreID:     shelf.StoreID,
		ZoneID:      shelf.ZoneID,
		Code:        shelf.Code,
		Name:        shelf.Name,
		Description: shelf.Description,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return shelf, nil
}

func (r *AdminRepository) DeleteShelf(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ShelfModel{}, id).Error
}

// ---------- Product ----------

func (r *AdminRepository) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	m := ProductModel{
		Name:     product.Name,
		Brand:    product.Brand,
		Category: product.Category,
		Aliases:  encodeJSONList(product.Aliases),
		Tags:     encodeJSONList(product.Tags),
		Status:   product.Status,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	product.ID = m.ID
	return product, nil
}

func (r *AdminRepository) UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	m := ProductModel{
		ID:       product.ID,
		Name:     product.Name,
		Brand:    product.Brand,
		Category: product.Category,
		Aliases:  encodeJSONList(product.Aliases),
		Tags:     encodeJSONList(product.Tags),
		Status:   product.Status,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func (r *AdminRepository) DeleteProduct(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ProductModel{}, id).Error
}

// ---------- SKU ----------

func (r *AdminRepository) CreateSKU(ctx context.Context, sku *domain.SKU) (*domain.SKU, error) {
	m := SKUModel{
		ProductID: sku.ProductID,
		Barcode:   sku.Barcode,
		Spec:      sku.Spec,
		Price:     sku.Price,
		Status:    sku.Status,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	sku.ID = m.ID
	return sku, nil
}

func (r *AdminRepository) UpdateSKU(ctx context.Context, sku *domain.SKU) (*domain.SKU, error) {
	m := SKUModel{
		ID:        sku.ID,
		ProductID: sku.ProductID,
		Barcode:   sku.Barcode,
		Spec:      sku.Spec,
		Price:     sku.Price,
		Status:    sku.Status,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return sku, nil
}

func (r *AdminRepository) DeleteSKU(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&SKUModel{}, id).Error
}

// ---------- ProductLocation ----------

func (r *AdminRepository) CreateProductLocation(ctx context.Context, pl *domain.ProductLocation) (*domain.ProductLocation, error) {
	m := ProductLocationModel{
		StoreID:      pl.StoreID,
		ProductID:    pl.ProductID,
		SKUID:        pl.SKUID,
		ZoneID:       pl.ZoneID,
		ShelfID:      pl.ShelfID,
		LayerNo:      pl.LayerNo,
		PositionDesc: pl.PositionDesc,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	pl.ID = m.ID
	return pl, nil
}

func (r *AdminRepository) UpdateProductLocation(ctx context.Context, pl *domain.ProductLocation) (*domain.ProductLocation, error) {
	m := ProductLocationModel{
		ID:           pl.ID,
		StoreID:      pl.StoreID,
		ProductID:    pl.ProductID,
		SKUID:        pl.SKUID,
		ZoneID:       pl.ZoneID,
		ShelfID:      pl.ShelfID,
		LayerNo:      pl.LayerNo,
		PositionDesc: pl.PositionDesc,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return pl, nil
}

func (r *AdminRepository) DeleteProductLocation(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ProductLocationModel{}, id).Error
}

// ---------- Inventory ----------

func (r *AdminRepository) CreateInventory(ctx context.Context, inv *domain.Inventory) (*domain.Inventory, error) {
	m := InventoryModel{
		StoreID:        inv.StoreID,
		SKUID:          inv.SKUID,
		Quantity:       inv.Quantity,
		SafetyStock:    inv.SafetyStock,
		LastVerifiedAt: inv.LastVerifiedAt,
		UpdateSource:   inv.UpdateSource,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	inv.ID = m.ID
	inv.UpdatedAt = m.UpdatedAt
	return inv, nil
}

func (r *AdminRepository) UpdateInventory(ctx context.Context, inv *domain.Inventory) (*domain.Inventory, error) {
	m := InventoryModel{
		ID:             inv.ID,
		StoreID:        inv.StoreID,
		SKUID:          inv.SKUID,
		Quantity:       inv.Quantity,
		SafetyStock:    inv.SafetyStock,
		LastVerifiedAt: inv.LastVerifiedAt,
		UpdateSource:   inv.UpdateSource,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	inv.UpdatedAt = m.UpdatedAt
	return inv, nil
}

func (r *AdminRepository) DeleteInventory(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&InventoryModel{}, id).Error
}

// ---------- Promotion ----------

func (r *AdminRepository) CreatePromotion(ctx context.Context, promo *domain.Promotion) (*domain.Promotion, error) {
	m := PromotionModel{
		StoreID:      promo.StoreID,
		Title:        promo.Title,
		Description:  promo.Description,
		ProductScope: encodeJSONList(promo.ProductScope),
		StartAt:      promo.StartAt,
		EndAt:        promo.EndAt,
		Status:       promo.Status,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	promo.ID = m.ID
	return promo, nil
}

func (r *AdminRepository) UpdatePromotion(ctx context.Context, promo *domain.Promotion) (*domain.Promotion, error) {
	m := PromotionModel{
		ID:           promo.ID,
		StoreID:      promo.StoreID,
		Title:        promo.Title,
		Description:  promo.Description,
		ProductScope: encodeJSONList(promo.ProductScope),
		StartAt:      promo.StartAt,
		EndAt:        promo.EndAt,
		Status:       promo.Status,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return promo, nil
}

func (r *AdminRepository) DeletePromotion(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&PromotionModel{}, id).Error
}

// ---------- FAQ ----------

func (r *AdminRepository) CreateFAQ(ctx context.Context, faq *domain.FAQ) (*domain.FAQ, error) {
	m := FAQModel{
		StoreID:  faq.StoreID,
		Question: faq.Question,
		Answer:   faq.Answer,
		Category: faq.Category,
		Keywords: encodeJSONList(faq.Keywords),
		Status:   faq.Status,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	faq.ID = m.ID
	return faq, nil
}

func (r *AdminRepository) UpdateFAQ(ctx context.Context, faq *domain.FAQ) (*domain.FAQ, error) {
	m := FAQModel{
		ID:       faq.ID,
		StoreID:  faq.StoreID,
		Question: faq.Question,
		Answer:   faq.Answer,
		Category: faq.Category,
		Keywords: encodeJSONList(faq.Keywords),
		Status:   faq.Status,
	}
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return faq, nil
}

func (r *AdminRepository) DeleteFAQ(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&FAQModel{}, id).Error
}

// ---------- helpers ----------

func encodeJSONList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(raw)
}
