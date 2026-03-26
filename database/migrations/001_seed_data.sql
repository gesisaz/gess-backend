-- Migration 001: seed data (forward-only). Apply after database/schema.sql.

-- ============================================
-- SEED DATA: Cosmetics Store
-- ============================================

-- Insert cosmetics brands
INSERT INTO brands (name, description, country_of_origin, website_url, logo_url) VALUES
    ('Scent Theory', 'Premium natural skincare focusing on aromatherapy and wellness. Our products combine ancient botanical wisdom with modern science.', 'USA', 'https://scenttheory.com', 'https://myscenttheory.com/cdn/shop/files/SCENT_THEORY_LOGO_1_d9a3d3ad-94bb-4dc3-9dab-75baf1c5de97_180x.png?v=1614761242'),
    ('Dove', 'Personal care and beauty brand by Unilever.', 'USA', 'https://www.dove.com', 'https://www.unilever.com/content-images/92ui5egz/production/31ede91fba0d37ba4097099380ddf098c12c22d1-1080x1080.jpg?w=160&h=160&fit=crop&auto=format'),
    ('eos', 'Evolution of Smooth - lip balm and skincare.', 'USA', 'https://evolutionofsmooth.com', 'https://evolutionofsmooth.com/cdn/shop/files/logo_110px.png?v=1748979951'),
    ('Vaseline', 'Healing jelly and skincare by Unilever.', 'USA', 'https://www.vaseline.com', 'https://assets.unileversolutions.com/v1/1228921.png'),
    ('BOB AND BRAD', 'Physical therapy and wellness products.', 'USA', NULL, 'https://s3-eu-west-1.amazonaws.com/tpd/logos/62a94c86bebe68974e69149b/0x0.png'),
    ('Bionike', 'Italian dermocosmetic brand focused on skin care and wellbeing.', 'Italy', NULL, 'https://www.bionike.it/static/media/logo-regular.43ddfc0007988a7313cd5469519174f7.svg');

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
    is_featured, image_url, image_urls
) VALUES
(
    'Cashmere Skin',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'Roasted coconut, salted amber & sandalwood', ARRAY['all'], 'body',
    true,
    'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Cashmere-Skin-18oz_cd4dd774-08d3-4b2d-894c-87e5fe3e3239.4f3062d484282285fcf0d6ca46c2602d.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
    ARRAY[
        'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Cashmere-Skin-18oz_cd4dd774-08d3-4b2d-894c-87e5fe3e3239.4f3062d484282285fcf0d6ca46c2602d.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/7eb564ba-727b-47b3-9bba-a7c8cc8200f2.4ed5a340a6ea05bdcd8e33ee3d0eb5ac.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/22381f4b-72a6-43af-b29f-0a6266361c4c.c47fbbf8bff962e30865f1d24eb0c54d.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/25d66320-8578-45da-bdfb-2b112cfb6ba6.99bc0826ed6302b04f0351e2f590850c.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/f8f3d0c2-d023-4cd9-8f0c-d35772c497ed.7d8f5eb7c1c45b64a666ea2dbc6267ee.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/e4373adf-741a-4e75-aca0-c546bb4aefab.9b0ce3cf708e8e2fae0e374993eb6649.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/6a58a3b6-6e8b-46bb-9106-6f3ea8ec56b9.83d23a8f04418dc9844d9bf0d14ba029.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/d0397731-99ab-41ec-89f1-2f86f41c7750.30f6138dfe7872aa12aa28b500087642.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/ed0e1a8f-2660-4e41-bb7f-a35c43c30864.495ebef7853a2a4b385063a0417af455.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/c7e19212-0304-48f2-a570-0038e9461070.2e2732aee2ee683c3010e213d4439e2b.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF'
    ]
),
(
    'Silk Sheets',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'lavender sprigs, fresh pear, creamy orchid', ARRAY['all'], 'body',
    true,
    'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Silk-Sheets-18oz_e8ccd455-dbb3-4fee-8447-8ded0aaa12a2.516091dbe960162747ea09724696f530.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
    ARRAY[
        'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Silk-Sheets-18oz_e8ccd455-dbb3-4fee-8447-8ded0aaa12a2.516091dbe960162747ea09724696f530.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/99ecb1e0-051f-470a-873a-25feb20cdd1f.2e8051ea1f2b1e6eee8d8bfd1beb076d.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/6fced69c-ee7e-4c3d-a019-779fbb912d90.c3eefffaf3c8c84f9f3ad435097a39ff.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/25d66320-8578-45da-bdfb-2b112cfb6ba6.99bc0826ed6302b04f0351e2f590850c.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/57f56953-b2e3-44ec-87d3-df80ea9cfd43.4e4a75e2b39e902c3b7222d993a3c896.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/65de39d2-2e89-4275-adec-a917d5506e25.d65ebefed514507da45ce5727b4b88d0.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/54f4a724-dc2a-4800-a7df-2526f340a5d3.4062b301197b9d6c4bda1f489b105b9a.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/9d9b2044-ee08-4c43-a888-c6436558b333.6ccbb660e0aa16236a9ecf20ab8cb648.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/426d4040-5f6f-4244-9b9d-10d7a51a6e01.9534d042d370b86c88fc282f8411ec45.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/b4df3320-6290-419a-bd8f-2811090f1fb7.4948fcabe4eeb7317385651fcb3449eb.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF'
    ]
),
(
    'Velvet Vanilla',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'Whipped Buttercream, Caramelized Sugar, Vanilla', ARRAY['all'], 'body',
    true,
    'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Velvet-Vanilla-18oz_56823db9-c0a4-4bea-95bd-d97b7c1408ac.768123ee0d29a02e71ad56cc2e379285.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
    ARRAY[
        'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Velvet-Vanilla-18oz_56823db9-c0a4-4bea-95bd-d97b7c1408ac.768123ee0d29a02e71ad56cc2e379285.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/b1b563e7-28ca-40c6-8e8c-632b808c3fdb.2403110fb83eb5f6c337f5bc4ab5a78a.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/62d458b9-94f0-4e1c-a72c-0f3c7f10d7f8.ac9688df30935aa8bcb983dabb2a40bf.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/25d66320-8578-45da-bdfb-2b112cfb6ba6.99bc0826ed6302b04f0351e2f590850c.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/4d003f84-561b-4b91-b08e-24b3964b0112.cc9ab16f21f9e7bee9ee1760f38f9de0.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/65ab2a8d-d644-40fb-8715-3e6bb9e984c2.1914e5e491fc2d91d3f813de873f2906.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/13210cc5-2114-40c2-ba39-ebcbea9cf4b3.d2e3c171085aa0e2ecc7261b9f9e86a2.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/83b628ac-3497-4ebf-91b9-fb9ce149df8e.05bb1c6c553b515d2ffa0602bb3bf722.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/ef2f20cf-41f0-4664-854e-54b2352e7825.cdb49c5efd87e1fe134fe6e7a3e3d1a1.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/f77dfb55-469d-474a-abb8-9bd49871bb36.b8e75e8fa3f1b52f27d054e6563ad64f.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF'
    ]
),
(
    'Linen Drift',
    NULL,
    945.00, 2500.00, 10,
    (SELECT id FROM brands WHERE name = 'Scent Theory'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    532, 'ml', 'fresh air, sun-kissed honeysuckle & cool cotton', ARRAY['all'], 'body',
    true,
    'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Washed-Linen-18oz_ca02112d-1744-4ba2-9351-28e8af56629b.fca21744a7e08dfd31f95e170d853275.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
    ARRAY[
        'https://i5.walmartimages.com/seo/Scent-Theory-Fragrance-Body-Lotion-Washed-Linen-18oz_ca02112d-1744-4ba2-9351-28e8af56629b.fca21744a7e08dfd31f95e170d853275.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/2677e95d-f7f6-47f4-b4c2-d13bfae29ce7.72de1634667d8e8162213646273b286e.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/1be3b8d0-aaf4-4481-890d-2b880c9131ea.077e69e9cccf8894acbbad67a6d60e6c.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/6d7f459a-c09d-42ea-9d00-cbf793c8ae9d.37079507eb9fd2d166aa9eb6dbc3040c.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/25d66320-8578-45da-bdfb-2b112cfb6ba6.99bc0826ed6302b04f0351e2f590850c.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/fc6cc060-1741-4d49-9213-4310bf7d5d0a.08872bbbd4cf82f3669a2405b8df88da.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/79d3a320-ac18-4020-ab4a-3b46f8241e58.e05d52c2b2b023e674c8b07dea0e2b14.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/187f28cf-ebfb-4559-ae64-d8bc7e256218.7cfc0c4ccf574aa0b126177e8460b530.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/50ba6e17-f21b-4050-8dc1-5ec6dd396c62.1b996b34417d63da7b95dd0f7b58cc70.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/2df2138b-b246-4a8a-87a4-945a5d5cd8e7.0fffe32a5d6457307c447a22dc647740.png?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF'
    ]
),
(
    'Vaseline Intensive Care Cocoa Radiant Body Gel Oil for Glowing Skin, 6.8 oz',
    NULL,
    0.00, 2000.00, 5,
    (SELECT id FROM brands WHERE name = 'Vaseline'),
    (SELECT id FROM categories WHERE name = 'Body Oils'),
    200, 'ml', NULL, ARRAY['all'], 'body',
    false,
    'https://assets.unileversolutions.com/v1/80751678.png?im=Resize,width=985,height=985',
    ARRAY['https://assets.unileversolutions.com/v1/80751678.png?im=Resize,width=985,height=985']
),
(
    'Vaseline Hand Cream for Dry Skin - Hydra Strength, 3.4 Oz Ea (Pack of 2)',
    NULL,
    289.00, 1500.00, 2,
    (SELECT id FROM brands WHERE name = 'Vaseline'),
    (SELECT id FROM categories WHERE name = 'Hand Creams'),
    100, 'ml', NULL, ARRAY['dry'], 'hands',
    false,
    'https://i5.walmartimages.com/asr/5d67d1ae-d276-495a-a03b-8c47aef4e6ce.9251fee139ffde5c2f27a415656ff41c.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
    ARRAY[
        'https://i5.walmartimages.com/asr/5d67d1ae-d276-495a-a03b-8c47aef4e6ce.9251fee139ffde5c2f27a415656ff41c.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/d4fe0cc2-7338-4012-90bb-04def49812fd.c6b3475a3b712abc58eced48f0b69ca0.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/95a9d5a4-723c-4454-9230-3e88d5d9487f.71d345e6dbd6aab1d4f0518bd4c8518a.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/a2dce948-8c22-4432-be2c-1cd0b144141f.3d2ca39c652d21e6fdd5debc9c0bd0db.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/3f8726db-628f-4443-97cd-526d540112e1.b0d3052d89e5e3f0d35e0636ca20f010.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/67273b8e-6fb3-4174-8236-92a5de486671.49afcecbc75f94628d46a9b066342748.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/a96b134f-1251-4702-a953-dd32893cf68a.64d5208004c65991b5aaa9ad16859a40.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/bd8554e6-8ed9-44de-8427-27bc10ba6151.fce86f2a570ef69aa3026923a56cd85e.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF'
    ]
),
(
    'Vaseline Original Petroleum Jelly, 7.5 oz',
    NULL,
    400.00, 600.00, 5,
    (SELECT id FROM brands WHERE name = 'Vaseline'),
    (SELECT id FROM categories WHERE name = 'Body Care'),
    212, 'g', NULL, ARRAY['all'], 'body',
    false,
    'https://i5.walmartimages.com/seo/Vaseline-Original-Petroleum-Jelly-7-5-oz_30908ec9-2389-4f48-99ad-44c6018b56f3.2ab4090ad3c280802f92fe3d62ea7bda.jpeg?odnHeight=573&odnWidth=573&odnBg=FFFFFF',
    ARRAY[
        'https://i5.walmartimages.com/seo/Vaseline-Original-Petroleum-Jelly-7-5-oz_30908ec9-2389-4f48-99ad-44c6018b56f3.2ab4090ad3c280802f92fe3d62ea7bda.jpeg?odnHeight=573&odnWidth=573&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/20a9a3a6-6953-4df0-81bc-52564a0a5445.f3dcf7baa8e1235c805c9b3b0dacd73f.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/ee201e9b-3b49-4139-adb2-dfa1b89c94cd.00d14a363fdeb1f9d8ed4f4c653b8872.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/28c65e12-f801-4af8-9acf-73ad77f42ef6.e347f08a585af9e7871d12bb0153a475.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/98bf235c-3b24-472f-824f-22ddeba0f499.e203f3330dfd12d35ce4ef75d204f0bd.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/791d88c7-093a-43be-8313-4663dfc20ee2.33abae78df0288e9e853a51453e96f57.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/fd6916c7-f0ce-4928-89b7-10c9276fdd00.a15fd9fa6b225b3962ac49518c65edca.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF',
        'https://i5.walmartimages.com/asr/984b91ad-e87b-42c3-bff5-18191270cb74.5e58c8064817242378fe9df4f59e7563.jpeg?odnHeight=2000&odnWidth=2000&odnBg=FFFFFF'
    ]
),
(
    'Weighted Heating Pad with Far Infrared Therapy',
    NULL,
    3868.00, 5000.00, 2,
    (SELECT id FROM brands WHERE name = 'BOB AND BRAD'),
    (SELECT id FROM categories WHERE name = 'Wellness'),
    2.4, 'lbs', NULL, NULL, NULL,
    false,
    'https://m.media-amazon.com/images/I/81CV0gNeC0L._AC_SL1500_.jpg',
    ARRAY[
        'https://m.media-amazon.com/images/I/81CV0gNeC0L._AC_SL1500_.jpg',
        'https://m.media-amazon.com/images/I/81K4uIWytbL._AC_SL1500_.jpg',
        'https://m.media-amazon.com/images/I/71creKubezL._AC_SL1500_.jpg',
        'https://m.media-amazon.com/images/I/61salb7zzJL._AC_SL1500_.jpg',
        'https://m.media-amazon.com/images/I/71COQzwOfML._AC_SL1500_.jpg',
        'https://m.media-amazon.com/images/I/71nrIsdyGAL._AC_SL1500_.jpg',
        'https://m.media-amazon.com/images/I/71sW0f1kZGL._AC_SL1500_.jpg'
    ]
),
(
    'Dove Soap Original',
    NULL,
    200.00, 350.00, 48,
    (SELECT id FROM brands WHERE name = 'Dove'),
    (SELECT id FROM categories WHERE name = 'Body Care'),
    135, 'g', NULL, NULL, NULL,
    false,
    'https://m.media-amazon.com/images/I/71DxBnMlxFL._SL1500_.jpg',
    ARRAY[
        'https://m.media-amazon.com/images/I/71DxBnMlxFL._SL1500_.jpg',
        'https://m.media-amazon.com/images/I/41dZqwXwJSL._SL1500_.jpg'
    ]
),
(
    'eos Shea Better Sensitive Skin Body Lotion for Dry Skin',
    NULL,
    300.00, 2000.00, 6,
    (SELECT id FROM brands WHERE name = 'eos'),
    (SELECT id FROM categories WHERE name = 'Body Lotions'),
    473, 'ml', 'Fragrance-Free', ARRAY['dry', 'sensitive'], 'body',
    false,
    'https://evolutionofsmooth.com/cdn/shop/files/file_52.jpg?v=1762353481',
    ARRAY[
        'https://evolutionofsmooth.com/cdn/shop/files/file_52.jpg?v=1762353481',
        'https://evolutionofsmooth.com/cdn/shop/files/021325_eos-JRU-766_4X5_62c36e2e-5301-4438-be78-edd41de0c2f6.jpg?v=1763671199',
        'https://evolutionofsmooth.com/cdn/shop/files/VanillaCashmereBL_2.jpg?v=1758024162',
        'https://evolutionofsmooth.com/cdn/shop/files/031125_eos-JRU-23-1000X1161.jpg?v=1758024162',
        'https://evolutionofsmooth.com/cdn/shop/files/021325_eos-JRU-693_4X5_264388a8-e7ff-48d8-823c-b51afcf8d66c.jpg?v=1758024162'
    ]
);

INSERT INTO users (username, email, password_hash, role) VALUES
    ('caroline', 'caroline@example.com', '$2y$10$CPtXXlzvYy9Y0ka/WNwg2.ebBeXSM.9eDBR.j0UeFcCkOWW1qnUL2', 'admin'),
    ('moseti', 'moseti@gess.com', '$2y$10$5dRqbiQtnm74JRO4AgSrbeQCjugH0EsFXtY4zrKeUNJF6jX8Z.th2', 'user');
