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
    email_verified_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Verification tokens (email verification, password reset)
CREATE TABLE verification_tokens (
    token VARCHAR(255) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_verification_tokens_token_type ON verification_tokens(token, type);

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
    buying_price NUMERIC(10, 2) NOT NULL CHECK (buying_price >= 0),
    selling_price NUMERIC(10, 2) NOT NULL CHECK (selling_price >= 0),
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
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    total_amount NUMERIC(10, 2) NOT NULL CHECK (total_amount >= 0),
    status order_status NOT NULL DEFAULT 'pending',
    shipping_address_id UUID REFERENCES addresses(id),
    guest_email VARCHAR(255),
    guest_name VARCHAR(255),
    shipping_full_name VARCHAR(255),
    shipping_street_address TEXT,
    shipping_city VARCHAR(100),
    shipping_state VARCHAR(100),
    shipping_postal_code VARCHAR(20),
    shipping_country VARCHAR(100),
    shipping_phone VARCHAR(20),
    guest_phone VARCHAR(20),
    guest_shipping_line1 TEXT,
    guest_shipping_line2 TEXT,
    guest_shipping_city VARCHAR(100),
    guest_shipping_postal_code VARCHAR(20),
    guest_shipping_country VARCHAR(100),
    mpesa_checkout_request_id VARCHAR(255),
    mpesa_merchant_request_id VARCHAR(255),
    mpesa_receipt_number VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT orders_owner_check CHECK (
        (user_id IS NOT NULL) OR (guest_email IS NOT NULL)
    )
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
CREATE INDEX idx_orders_mpesa_checkout ON orders(mpesa_checkout_request_id) WHERE mpesa_checkout_request_id IS NOT NULL;
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
    ('Luxe Beauty', 'High-end cosmetics and skincare for the discerning customer. Luxury that loves your skin.', 'France', 'https://luxebeauty.fr'),
    ('Dove', 'Personal care and beauty brand by Unilever.', 'USA', 'https://www.dove.com'),
    ('eos', 'Evolution of Smooth - lip balm and skincare.', 'USA', 'https://evolutionofsmooth.com'),
    ('Vaseline', 'Healing jelly and skincare by Unilever.', 'USA', 'https://www.vaseline.com'),
    ('BOB AND BRAD', 'Physical therapy and wellness products.', 'USA', NULL);

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
        (SELECT id FROM categories WHERE name = 'Hand Care'), 2),
    ('Wellness', 'Wellness and pain relief products',
        (SELECT id FROM categories WHERE name = 'Body Care'), 4);

INSERT INTO products (
    name, description, 
    buying_price, selling_price, stock_quantity,
    brand_id, category_id, 
    size_value, size_unit, scent, skin_type, application_area,
    image_url
) VALUES
(
    'Cashmere Skin',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'Roasted coconut, salted amber & sandalwood', ARRAY['all'], 'body',
    'https://myscenttheory.com/cdn/shop/files/Wrapped_Cashmere_01_600x.png?v=1765944539'
),
(
    'Silk Sheets',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'lavender sprigs, fresh pear, creamy orchid', ARRAY['all'], 'body',
    'https://myscenttheory.com/cdn/shop/files/Wrapped_Silk_01_600x.png?v=1765944280'
),
(
    'Velvet Vanilla',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'Whipped Buttercream, Caramelized Sugar, Vanilla', ARRAY['all'], 'body',
    'https://myscenttheory.com/cdn/shop/files/Wrapped_Vanilla_01_600x.png?v=1765944049'
),
(
    'Linen Drift',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'fresh air, sun-kissed honeysuckle & cool cotton', ARRAY['all'], 'body',
    'https://myscenttheory.com/cdn/shop/files/Wrapped_Linen_01_600x.png?v=1765944425'
),
(
    'Vaseline Intensive Care Cocoa Radiant Body Gel Oil for Glowing Skin, 6.8 oz',
    NULL,
    0.00, 2000.00, 5,
    (SELECT id FROM brands WHERE name = 'Vaseline'),
    (SELECT id FROM categories WHERE name = 'Body Oils'),
    200, 'ml', NULL, ARRAY['all'], 'body',
    'https://assets.unileversolutions.com/v1/80751678.png?im=Resize,width=985,height=985'
),
(
    'Vaseline Hand Cream for Dry Skin - Hydra Strength, 3.4 Oz Ea (Pack of 2)',
    NULL,
    289.00, 1500.00, 2,
    (SELECT id FROM brands WHERE name = 'Vaseline'),
    (SELECT id FROM categories WHERE name = 'Hand Creams'),
    100, 'ml', NULL, ARRAY['dry'], 'hands',
    'https://i5.walmartimages.com/seo/Vaseline-Hand-Cream-Dry-Skin-Hydra-Strength-Hand-Lotion-Hyaluronic-Acid-Vitamin-C-Shea-Butter-Intensive-Care-Hand-Repair-Cream-3-4-Oz-Ea-Pack-2_1feda659-b96a-44ce-a42b-d3868f4bd0fc.84f74945f9b96b7d4540c730320e9a8f.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF'
),
(
    'Vaseline Original Petroleum Jelly, 7.5 oz',
    NULL,
    400.00, 600.00, 5,
    (SELECT id FROM brands WHERE name = 'Vaseline'),
    (SELECT id FROM categories WHERE name = 'Body Care'),
    212, 'g', NULL, ARRAY['all'], 'body',
    'https://assets.unileversolutions.com/v1/1474787.png?im=Resize,width=985,height=985'
),
(
    'Weighted Heating Pad with Far Infrared Therapy',
    NULL,
    3868.00, 5000.00, 2,
    (SELECT id FROM brands WHERE name = 'BOB AND BRAD'),
    (SELECT id FROM categories WHERE name = 'Wellness'),
    2.4, 'lbs', NULL, NULL, NULL,
    'https://m.media-amazon.com/images/I/81wcqMtBKDL._AC_SL1500_.jpg'
),
(
    'Dove Soap Original',
    NULL,
    200.00, 350.00, 48,
    (SELECT id FROM brands WHERE name = 'Dove'),
    (SELECT id FROM categories WHERE name = 'Body Care'),
    135, 'g', NULL, NULL, NULL,
    'https://m.media-amazon.com/images/I/61net67nNYL._SL1500_.jpg'
),
(
    'eos Shea Better Sensitive Skin Body Lotion for Dry Skin',
    NULL,
    300.00, 2000.00, 6,
    (SELECT id FROM brands WHERE name = 'eos'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    473, 'ml', 'Fragrance-Free', ARRAY['dry', 'sensitive'], 'body',
    'https://evolutionofsmooth.com/cdn/shop/files/FragranceFreeBL.jpg?v=1758024313&width=900'
);

INSERT INTO users (username, email, password_hash, role) VALUES
    ('caroline', 'caroline@example.com', '$2y$10$CPtXXlzvYy9Y0ka/WNwg2.ebBeXSM.9eDBR.j0UeFcCkOWW1qnUL2', 'admin'),
    ('moseti', 'moseti@gess.com', '$2y$10$5dRqbiQtnm74JRO4AgSrbeQCjugH0EsFXtY4zrKeUNJF6jX8Z.th2', 'user');