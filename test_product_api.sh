#!/bin/bash

# Cosmetics E-commerce API Test Script
# Tests all product, brand, and category endpoints with cosmetics features

BASE_URL="http://localhost:8080"
COOKIE_FILE="cookies.txt"

echo "========================================="
echo "Cosmetics E-commerce API Testing"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASS=0
FAIL=0

# Function to print test result
test_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((PASS++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((FAIL++))
    fi
}

# Function to print section header
section_header() {
    echo ""
    echo -e "${BLUE}=========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=========================================${NC}"
}

# Clean up previous cookie file
rm -f $COOKIE_FILE

section_header "SETUP: User Registration"

echo "Step 1: Register a regular user"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "409" ] && echo 0 || echo 1) "Register user (got $HTTP_CODE)"

echo ""
echo "Step 2: Register an admin user"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"adminpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "409" ] && echo 0 || echo 1) "Register admin user (got $HTTP_CODE)"

echo ""
echo -e "${YELLOW}MANUAL STEP REQUIRED:${NC}"
echo "Run this command to promote the user to admin:"
echo "psql -d gess_staging -c \"UPDATE users SET role = 'admin' WHERE username = 'admin';\""
echo ""
read -p "Press Enter after promoting user to admin..."

section_header "AUTHENTICATION"

echo "Step 3: Login as admin"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"adminpass123"}' \
  -c $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Admin login (expected 200, got $HTTP_CODE)"

section_header "BRAND MANAGEMENT"

echo "Step 4: Create a brand (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/brands \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d '{
    "name": "Test Cosmetics Co.",
    "description": "A test cosmetics brand for API testing",
    "country_of_origin": "USA",
    "website_url": "https://testcosmetics.com"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "201" ] && echo 0 || echo 1) "Create brand (expected 201, got $HTTP_CODE)"
echo "Response: $BODY"
echo ""

# Extract brand ID
BRAND_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created brand ID: $BRAND_ID"

echo ""
echo "Step 5: List all brands (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" $BASE_URL/brands)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "List brands (expected 200, got $HTTP_CODE)"
echo "Found brands: $(echo "$BODY" | grep -o '"name"' | wc -l)"

echo ""
echo "Step 6: Get single brand with products (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/brands/$BRAND_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Get brand (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 7: Update brand (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/admin/brands/$BRAND_ID" \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d '{
    "description": "Updated test cosmetics brand"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Update brand (expected 200, got $HTTP_CODE)"

section_header "CATEGORY MANAGEMENT (Hierarchical)"

echo "Step 8: Create parent category (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/categories \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d '{
    "name": "Test Body Care",
    "description": "Test body care products",
    "display_order": 1
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "201" ] && echo 0 || echo 1) "Create parent category (expected 201, got $HTTP_CODE)"

PARENT_CATEGORY_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created parent category ID: $PARENT_CATEGORY_ID"

echo ""
echo "Step 9: Create child category (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/categories \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d "{
    \"name\": \"Test Body Lotions\",
    \"description\": \"Test body lotions subcategory\",
    \"parent_category_id\": \"$PARENT_CATEGORY_ID\",
    \"display_order\": 1
  }")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "201" ] && echo 0 || echo 1) "Create child category (expected 201, got $HTTP_CODE)"

CHILD_CATEGORY_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created child category ID: $CHILD_CATEGORY_ID"

echo ""
echo "Step 10: List categories hierarchically (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/categories?hierarchical=true")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "List hierarchical categories (expected 200, got $HTTP_CODE)"
echo "Response preview:" 
echo "$BODY" | head -c 300

echo ""
echo ""
echo "Step 11: List parent categories only (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/categories?parent_only=true")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "List parent categories (expected 200, got $HTTP_CODE)"

section_header "PRODUCT MANAGEMENT (Cosmetics)"

echo "Step 12: Create cosmetics product with all attributes (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d "{
    \"name\": \"Test Lavender Body Lotion\",
    \"description\": \"Ultra-hydrating test body lotion with lavender\",
    \"price\": 24.99,
    \"stock_quantity\": 100,
    \"brand_id\": \"$BRAND_ID\",
    \"category_id\": \"$CHILD_CATEGORY_ID\",
    \"sku\": \"TEST-BL-LAV-250\",
    \"product_line\": \"Test Collection\",
    \"size_value\": 250,
    \"size_unit\": \"ml\",
    \"scent\": \"Lavender Vanilla\",
    \"skin_type\": [\"dry\", \"normal\", \"sensitive\"],
    \"ingredients\": \"Aqua, Shea Butter, Lavender Oil, Vitamin E\",
    \"key_ingredients\": [\"Lavender Oil\", \"Shea Butter\", \"Vitamin E\"],
    \"application_area\": \"body\",
    \"is_organic\": true,
    \"is_vegan\": true,
    \"is_cruelty_free\": true,
    \"is_paraben_free\": true,
    \"is_featured\": true,
    \"image_url\": \"https://example.com/lavender.jpg\"
  }")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "201" ] && echo 0 || echo 1) "Create cosmetics product (expected 201, got $HTTP_CODE)"
echo "Response: $BODY" | head -c 300
echo ""

PRODUCT_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created product ID: $PRODUCT_ID"

echo ""
echo "Step 13: Create another product for filtering tests"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d "{
    \"name\": \"Test Rose Face Cream\",
    \"description\": \"Luxurious face cream with rose extract\",
    \"price\": 39.99,
    \"stock_quantity\": 50,
    \"brand_id\": \"$BRAND_ID\",
    \"category_id\": \"$CHILD_CATEGORY_ID\",
    \"sku\": \"TEST-FC-ROSE-50\",
    \"size_value\": 50,
    \"size_unit\": \"ml\",
    \"scent\": \"Rose\",
    \"skin_type\": [\"dry\", \"mature\"],
    \"application_area\": \"face\",
    \"is_organic\": false,
    \"is_vegan\": true,
    \"is_cruelty_free\": true,
    \"is_featured\": false
  }")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "201" ] && echo 0 || echo 1) "Create second product (expected 201, got $HTTP_CODE)"

PRODUCT_ID_2=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created second product ID: $PRODUCT_ID_2"

section_header "COSMETICS-SPECIFIC FILTERING"

echo "Step 14: Filter by brand (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?brand=$BRAND_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by brand (expected 200, got $HTTP_CODE)"
PRODUCT_COUNT=$(echo "$BODY" | grep -o '"id"' | wc -l)
echo "Found $PRODUCT_COUNT products for brand"

echo ""
echo "Step 15: Filter by skin type (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?skin_type=dry")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by skin type (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 16: Filter by application area (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?application_area=body")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by application area (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 17: Filter by scent (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?scent=lavender")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by scent (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 18: Filter by organic (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?is_organic=true")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by organic (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 19: Filter by vegan (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?is_vegan=true")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by vegan (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 20: Filter by cruelty-free (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?is_cruelty_free=true")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by cruelty-free (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 21: Filter by featured (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?is_featured=true")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Filter by featured (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 22: Multiple filters combined (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?is_organic=true&skin_type=dry&application_area=body&max_price=30")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Multiple filters (expected 200, got $HTTP_CODE)"

section_header "PRODUCT OPERATIONS"

echo "Step 23: Get single product with all cosmetics fields (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products/$PRODUCT_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Get product (expected 200, got $HTTP_CODE)"
echo "Product data:"
echo "$BODY" | head -c 400
echo ""

echo ""
echo "Step 24: Update product cosmetics fields (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/admin/products/$PRODUCT_ID" \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d '{
    "price": 22.99,
    "scent": "Lavender Dreams",
    "is_featured": false
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Update product (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 25: Search products (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?search=lavender")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Search products (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 26: Price range filter (public)"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products?min_price=20&max_price=30")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Price range filter (expected 200, got $HTTP_CODE)"

section_header "AUTHORIZATION TESTS"

echo "Step 27: Test non-admin access (should fail)"
# Login as regular user
curl -s -X POST $BASE_URL/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass123"}' \
  -c cookies_user.txt > /dev/null

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/brands \
  -H "Content-Type: application/json" \
  -b cookies_user.txt \
  -d '{
    "name": "Should Fail Brand"
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "403" ] && echo 0 || echo 1) "Non-admin create brand (expected 403, got $HTTP_CODE)"

echo ""
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b cookies_user.txt \
  -d '{
    "name": "Should Fail",
    "price": 100,
    "stock_quantity": 10
  }')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "403" ] && echo 0 || echo 1) "Non-admin create product (expected 403, got $HTTP_CODE)"

rm -f cookies_user.txt

section_header "VALIDATION TESTS"

echo "Step 28: Invalid SKU (should fail)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d "{
    \"name\": \"Test Invalid SKU\",
    \"price\": 10,
    \"stock_quantity\": 5,
    \"sku\": \"AB\"
  }")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "400" ] && echo 0 || echo 1) "Invalid SKU validation (expected 400, got $HTTP_CODE)"

echo ""
echo "Step 29: Invalid size unit (should fail)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d "{
    \"name\": \"Test Invalid Size\",
    \"price\": 10,
    \"stock_quantity\": 5,
    \"size_unit\": \"invalid\"
  }")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "400" ] && echo 0 || echo 1) "Invalid size unit validation (expected 400, got $HTTP_CODE)"

echo ""
echo "Step 30: Invalid skin type (should fail)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST $BASE_URL/admin/products \
  -H "Content-Type: application/json" \
  -b $COOKIE_FILE \
  -d "{
    \"name\": \"Test Invalid Skin Type\",
    \"price\": 10,
    \"stock_quantity\": 5,
    \"skin_type\": [\"invalid_type\"]
  }")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "400" ] && echo 0 || echo 1) "Invalid skin type validation (expected 400, got $HTTP_CODE)"

section_header "CLEANUP"

echo "Step 31: Delete products (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/admin/products/$PRODUCT_ID" -b $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Delete product 1 (expected 200, got $HTTP_CODE)"

echo ""
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/admin/products/$PRODUCT_ID_2" -b $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Delete product 2 (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 32: Verify product deletion"
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/products/$PRODUCT_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "404" ] && echo 0 || echo 1) "Verify deleted product (expected 404, got $HTTP_CODE)"

echo ""
echo "Step 33: Delete categories (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/admin/categories/$CHILD_CATEGORY_ID" -b $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Delete child category (expected 200, got $HTTP_CODE)"

echo ""
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/admin/categories/$PARENT_CATEGORY_ID" -b $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Delete parent category (expected 200, got $HTTP_CODE)"

echo ""
echo "Step 34: Deactivate brand (Admin)"
RESPONSE=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/admin/brands/$BRAND_ID" -b $COOKIE_FILE)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
test_result $([ "$HTTP_CODE" = "200" ] && echo 0 || echo 1) "Deactivate brand (expected 200, got $HTTP_CODE)"

# Clean up
rm -f $COOKIE_FILE

section_header "TEST SUMMARY"
echo ""
echo -e "${GREEN}Passed: $PASS${NC}"
echo -e "${RED}Failed: $FAIL${NC}"
echo "Total: $((PASS + FAIL))"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo ""
    echo "Your cosmetics e-commerce API is working perfectly!"
    exit 0
else
    echo -e "${RED}✗ Some tests failed.${NC}"
    echo ""
    echo "Please review the failures above."
    exit 1
fi
