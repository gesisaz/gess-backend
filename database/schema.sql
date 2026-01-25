-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create user role enum
CREATE TYPE user_role AS ENUM ('user', 'admin', 'super_admin');

-- Create order status enum
CREATE TYPE order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded');

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Brands table (cosmetics brands/companies)
CREATE TABLE brands (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    logo_url TEXT,
    website_url VARCHAR(255),
    country_of_origin VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Categories table (with hierarchy support)
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    parent_category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Products table (with cosmetics-specific fields)
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0),
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    
    -- Relationships
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    brand_id UUID REFERENCES brands(id) ON DELETE CASCADE,
    
    -- Product identification
    sku VARCHAR(100) UNIQUE,
    product_line VARCHAR(255),
    
    -- Size/Volume
    size_value NUMERIC(10, 2),
    size_unit VARCHAR(20), -- ml, oz, g, kg
    
    -- Cosmetics-specific attributes
    scent VARCHAR(255),
    skin_type TEXT[], -- Array: dry, oily, sensitive, normal, combination, all
    ingredients TEXT,
    key_ingredients TEXT[], -- Array of highlighted ingredients
    application_area VARCHAR(50), -- body, face, hands, feet, hair, nails
    
    -- Certifications/Features
    is_organic BOOLEAN DEFAULT false,
    is_vegan BOOLEAN DEFAULT false,
    is_cruelty_free BOOLEAN DEFAULT false,
    is_paraben_free BOOLEAN DEFAULT false,
    is_featured BOOLEAN DEFAULT false,
    
    -- Reviews
    rating_average NUMERIC(3, 2) DEFAULT 0 CHECK (rating_average >= 0 AND rating_average <= 5),
    review_count INTEGER DEFAULT 0,
    
    -- Media
    image_url TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Addresses table
CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    street_address TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Carts table
CREATE TABLE carts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Cart items table
CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cart_id, product_id)
);

-- Orders table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_amount NUMERIC(10, 2) NOT NULL CHECK (total_amount >= 0),
    status order_status NOT NULL DEFAULT 'pending',
    shipping_address_id UUID NOT NULL REFERENCES addresses(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Order items table
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price_at_purchase NUMERIC(10, 2) NOT NULL CHECK (price_at_purchase >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Reviews table
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    title VARCHAR(255),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(product_id, user_id)
);

-- Create indexes for better query performance

-- User indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);

-- Brand indexes
CREATE INDEX idx_brands_name ON brands(name);
CREATE INDEX idx_brands_active ON brands(is_active);

-- Category indexes
CREATE INDEX idx_categories_parent ON categories(parent_category_id);
CREATE INDEX idx_categories_display_order ON categories(display_order);

-- Product indexes
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_brand ON products(brand_id);
CREATE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_products_scent ON products(scent);
CREATE INDEX idx_products_featured ON products(is_featured);
CREATE INDEX idx_products_rating ON products(rating_average);
CREATE INDEX idx_products_application_area ON products(application_area);

-- Address indexes
CREATE INDEX idx_addresses_user ON addresses(user_id);

-- Cart indexes
CREATE INDEX idx_cart_items_cart ON cart_items(cart_id);
CREATE INDEX idx_cart_items_product ON cart_items(product_id);

-- Order indexes
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_product ON order_items(product_id);

-- Review indexes
CREATE INDEX idx_reviews_product ON reviews(product_id);
CREATE INDEX idx_reviews_user ON reviews(user_id);

-- ============================================
-- SEED DATA: Cosmetics Store
-- ============================================

-- Insert cosmetics brands
INSERT INTO brands (name, description, country_of_origin, website_url) VALUES
    ('Scent Theory', 'Premium natural skincare focusing on aromatherapy and wellness. Our products combine ancient botanical wisdom with modern science.', 'USA', 'https://scenttheory.com'),
    ('Pure Botanics', 'Certified organic skincare made from sustainably sourced ingredients. 100% natural, 100% effective.', 'Canada', 'https://purebotanics.ca'),
    ('Luxe Beauty', 'High-end cosmetics and skincare for the discerning customer. Luxury that loves your skin.', 'France', 'https://luxebeauty.fr');

-- Insert parent categories
INSERT INTO categories (name, description, display_order) VALUES
    ('Body Care', 'Complete body skincare and moisturizing products', 1),
    ('Face Care', 'Facial skincare, serums, and treatments', 2),
    ('Hand Care', 'Hand creams, lotions, and treatments', 3),
    ('Hair Care', 'Hair care and styling products', 4);

-- Insert subcategories
INSERT INTO categories (name, description, parent_category_id, display_order) VALUES
    -- Body Care subcategories
    ('Body Lotions', 'Daily moisturizing body lotions', 
        (SELECT id FROM categories WHERE name = 'Body Care'), 1),
    ('Body Butters', 'Rich, intensive body butters', 
        (SELECT id FROM categories WHERE name = 'Body Care'), 2),
    ('Body Oils', 'Nourishing body oils', 
        (SELECT id FROM categories WHERE name = 'Body Care'), 3),
    
    -- Face Care subcategories
    ('Face Moisturizers', 'Daily facial moisturizers', 
        (SELECT id FROM categories WHERE name = 'Face Care'), 1),
    ('Face Serums', 'Concentrated facial serums', 
        (SELECT id FROM categories WHERE name = 'Face Care'), 2),
    ('Face Masks', 'Intensive treatment masks', 
        (SELECT id FROM categories WHERE name = 'Face Care'), 3),
    
    -- Hand Care subcategories
    ('Hand Creams', 'Intensive hand creams', 
        (SELECT id FROM categories WHERE name = 'Hand Care'), 1),
    ('Hand Lotions', 'Light daily hand lotions', 
        (SELECT id FROM categories WHERE name = 'Hand Care'), 2);

-- Insert sample products from Scent Theory
INSERT INTO products (
    name, description, price, stock_quantity,
    brand_id, category_id, sku, product_line,
    size_value, size_unit, scent, skin_type, application_area,
    ingredients, key_ingredients,
    is_organic, is_vegan, is_cruelty_free, is_paraben_free, is_featured,
    image_url
) VALUES
-- Scent Theory Body Lotions
(
    'Lavender Dreams Body Lotion',
    'Ultra-hydrating body lotion infused with pure lavender essential oil and organic shea butter. Perfect for evening relaxation and deep moisturization. Our signature formula absorbs quickly without leaving a greasy residue.',
    24.99,
    150,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    'ST-BL-LAV-250',
    'Zen Collection',
    250,
    'ml',
    'Lavender Vanilla',
    ARRAY['dry', 'normal', 'sensitive'],
    'body',
    'Aqua, Butyrospermum Parkii (Shea) Butter, Lavandula Angustifolia (Lavender) Oil, Cocos Nucifera (Coconut) Oil, Tocopherol (Vitamin E), Aloe Barbadensis Leaf Juice',
    ARRAY['Lavender Essential Oil', 'Organic Shea Butter', 'Vitamin E', 'Aloe Vera'],
    true,
    true,
    true,
    true,
    true,
    'https://images.unsplash.com/photo-1608248543803-ba4f8c70ae0b?w=400'
),
(
    'Citrus Burst Body Lotion',
    'Energizing body lotion with orange and grapefruit extracts. Lightweight formula perfect for daily use. Awakens your senses while deeply nourishing your skin with natural botanicals.',
    24.99,
    200,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    'ST-BL-CIT-250',
    'Energize Collection',
    250,
    'ml',
    'Citrus Burst',
    ARRAY['all'],
    'body',
    'Aqua, Citrus Sinensis (Orange) Extract, Citrus Paradisi (Grapefruit) Oil, Cocos Nucifera (Coconut) Oil, Sodium Hyaluronate, Glycerin',
    ARRAY['Orange Extract', 'Grapefruit Oil', 'Coconut Oil', 'Hyaluronic Acid'],
    true,
    true,
    true,
    true,
    true,
    'https://images.unsplash.com/photo-1556228720-195a672e8a03?w=400'
),
(
    'Rose Petal Body Lotion',
    'Luxurious body lotion with Bulgarian rose extract and argan oil. Indulge in the romantic scent of fresh roses while treating your skin to intensive hydration and anti-aging benefits.',
    29.99,
    100,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    'ST-BL-RSE-500',
    'Romance Collection',
    500,
    'ml',
    'Rose Petal',
    ARRAY['dry', 'normal', 'mature'],
    'body',
    'Aqua, Rosa Damascena (Rose) Extract, Argania Spinosa (Argan) Oil, Ascorbic Acid (Vitamin C), Retinol, Aloe Barbadensis Leaf Juice',
    ARRAY['Bulgarian Rose Extract', 'Argan Oil', 'Vitamin C', 'Retinol'],
    true,
    true,
    true,
    true,
    false,
    'https://images.unsplash.com/photo-1615397349754-cfa2066a298e?w=400'
),

-- Pure Botanics Products
(
    'Eucalyptus Mint Body Butter',
    'Rich, intensive body butter with eucalyptus and peppermint. Provides long-lasting moisture for extremely dry skin. Certified organic and sustainably sourced ingredients.',
    34.99,
    80,
    (SELECT id FROM brands WHERE name = 'Pure Botanics'),
    (SELECT id FROM categories WHERE name = 'Body Butters'),
    'PB-BB-EUC-200',
    'Refresh Collection',
    200,
    'g',
    'Eucalyptus Mint',
    ARRAY['dry', 'very dry'],
    'body',
    'Butyrospermum Parkii (Shea) Butter, Eucalyptus Globulus Oil, Mentha Piperita (Peppermint) Oil, Theobroma Cacao (Cocoa) Butter, Tocopherol',
    ARRAY['Eucalyptus Oil', 'Peppermint Oil', 'Shea Butter', 'Cocoa Butter'],
    true,
    true,
    true,
    true,
    true,
    'https://images.unsplash.com/photo-1607748862156-7c548e7e98f4?w=400'
),
(
    'Vanilla Almond Hand Cream',
    'Deeply nourishing hand cream with sweet almond oil and vanilla. Non-greasy formula absorbs instantly. Perfect for frequent hand washing.',
    16.99,
    200,
    (SELECT id FROM brands WHERE name = 'Pure Botanics'),
    (SELECT id FROM categories WHERE name = 'Hand Creams'),
    'PB-HC-VAN-75',
    'Daily Essentials',
    75,
    'ml',
    'Vanilla Almond',
    ARRAY['all'],
    'hands',
    'Aqua, Prunus Amygdalus Dulcis (Almond) Oil, Vanilla Planifolia Extract, Glycerin, Butyrospermum Parkii Butter',
    ARRAY['Sweet Almond Oil', 'Vanilla Extract', 'Shea Butter'],
    true,
    true,
    true,
    true,
    false,
    'https://images.unsplash.com/photo-1556228578-8c89e6adf883?w=400'
),

-- Luxe Beauty Products
(
    'Gold Radiance Face Serum',
    'Luxury anti-aging serum with 24k gold flakes and hyaluronic acid. Visibly reduces fine lines and boosts skin radiance. Suitable for all skin types.',
    89.99,
    50,
    (SELECT id FROM brands WHERE name = 'Luxe Beauty'),
    (SELECT id FROM categories WHERE name = 'Face Serums'),
    'LB-FS-GLD-30',
    'Prestige Collection',
    30,
    'ml',
    'Unscented',
    ARRAY['all'],
    'face',
    'Aqua, Sodium Hyaluronate, Gold (24k), Retinol, Niacinamide, Peptide Complex, Vitamin C',
    ARRAY['24K Gold', 'Hyaluronic Acid', 'Retinol', 'Peptide Complex'],
    false,
    false,
    true,
    true,
    true,
    'https://images.unsplash.com/photo-1620916566398-39f1143ab7be?w=400'
),
(
    'Caviar Moisture Face Cream',
    'Ultra-luxe moisturizer with caviar extract and marine collagen. Provides intensive hydration and anti-aging benefits. The ultimate in facial care luxury.',
    124.99,
    30,
    (SELECT id FROM brands WHERE name = 'Luxe Beauty'),
    (SELECT id FROM categories WHERE name = 'Face Moisturizers'),
    'LB-FM-CAV-50',
    'Prestige Collection',
    50,
    'ml',
    'Unscented',
    ARRAY['normal', 'dry', 'mature'],
    'face',
    'Aqua, Caviar Extract, Hydrolyzed Marine Collagen, Hyaluronic Acid, Squalane, Ceramides, Peptides',
    ARRAY['Caviar Extract', 'Marine Collagen', 'Ceramides', 'Peptides'],
    false,
    false,
    true,
    true,
    true,
    'https://images.unsplash.com/photo-1570194065650-d99fb4bedf0a?w=400'
);

-- Update product ratings (simulated customer reviews)
UPDATE products SET rating_average = 4.8, review_count = 127 WHERE sku = 'ST-BL-LAV-250';
UPDATE products SET rating_average = 4.6, review_count = 89 WHERE sku = 'ST-BL-CIT-250';
UPDATE products SET rating_average = 4.9, review_count = 156 WHERE sku = 'ST-BL-RSE-500';
UPDATE products SET rating_average = 4.7, review_count = 94 WHERE sku = 'PB-BB-EUC-200';
UPDATE products SET rating_average = 4.5, review_count = 203 WHERE sku = 'PB-HC-VAN-75';
UPDATE products SET rating_average = 4.9, review_count = 67 WHERE sku = 'LB-FS-GLD-30';
UPDATE products SET rating_average = 5.0, review_count = 45 WHERE sku = 'LB-FM-CAV-50';
