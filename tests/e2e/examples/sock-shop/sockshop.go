// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sock_shop

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"io"
	"net/http"
	"strconv"
	"strings"

	. "github.com/onsi/gomega"
)

const (
	Debug = pkg.Debug
	Info  = pkg.Info
	Error = pkg.Error
)

// SockShop encapsulates all testing information related to interactions with the sock shop app
type SockShop struct {
	Cookies  	[]*http.Cookie
	Ingress  	string
	username 	string
	password 	string
	hostHeader 	string
}

// CatalogItem represents the information for an item in the catalog
type CatalogItem struct {
	Count       int      `json:"count"`
	Description string   `json:"description"`
	Id          string   `json:"id"`
	ImageUrl    string   `json:"imageurl"`
	Name        string   `json:"name"`
	Price       float32  `json:"price"`
	Tag         []string `json:"tag"`
}

// Catalog provides the contents of the catalog
type Catalog struct {
	Item []CatalogItem
}

// CartItem provides the details for an item in the cart
type CartItem struct {
	ItemId    string  `json:"itemId"`
	Quantity  int     `json:"quantity"`
	UnitPrice float32 `json:"unitPrice"`
}

// Cart contains the items of a cart
type Cart struct {
	Item []CartItem
}

// ID is an id struct
type Id struct {
	id string
}

// NewSockShop creates a new sockshop instance
func NewSockShop(username, password, ingress string, hostHeader string) SockShop {
	var sockShop SockShop
	sockShop.username = username
	sockShop.password = password
	sockShop.Ingress = ingress
	sockShop.hostHeader = hostHeader
	return sockShop
}

// Post is a wrapper function for HTTP request with cookies POST
func (s *SockShop) Post(url, contentType string, body io.Reader) (int, string) {
	return pkg.PostWithHostHeader(url, contentType, s.hostHeader, body)
}

// Get is a wrapper function for HTTP request with cookies GET
func (s *SockShop) Get(url string) (int, string) {
	return pkg.GetWebPageWithCABundle(url, s.hostHeader)
}

// Delete is a wrapper function for HTTP request with cookies DELETE
func (s *SockShop) Delete(url string) (int, string) {
	return pkg.Delete(url, s.hostHeader)
}

// RegisterUser interacts with sock shop to create a user
func (s *SockShop) RegisterUser(body string) {
	ingress := s.Ingress
	url := fmt.Sprintf("http://%v/register", ingress)
	status, register := pkg.PostWithHostHeader(url,"application/json", s.hostHeader, strings.NewReader(body))
	pkg.Log(Info, fmt.Sprintf("Finished register %v status: %v", register, status))
	Expect(status).To(Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", ingress, status))
	Expect(strings.Contains(register, "username")).To(Equal(true), fmt.Sprintf("Cannot register %v", register))
}

// ConnectToCatalog connects to the catalog page
func (s *SockShop) ConnectToCatalog() string {
	// connect to catalog
	ingress := s.Ingress
	pkg.Log(Info, fmt.Sprint("Connecting to Catalog"))
	status, webpage := s.Get("http://" + ingress + "/catalogue")
	Expect(status).To(Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", ingress, status))
	Expect(strings.Contains(webpage, "/catalogue/")).To(Equal(true), fmt.Sprintf("Webpage found is NOT the Catalog %v", webpage))
	return webpage
}

// VerifyCatalogItems gets catalog items and accesses detail page
func (s *SockShop) VerifyCatalogItems(webpage string) {
	//ingress := s.Ingress
	var items []CatalogItem
	json.Unmarshal([]byte(webpage), &items)
	Expect(len(items)).To(Not(Equal(0)), fmt.Sprint("Catalog page returned no items"))
}

// GetCatalogItem retrieves the first catalog item
func (s *SockShop) GetCatalogItem() Catalog {
	ingress := s.Ingress
	pkg.Log(Info, fmt.Sprint("Connecting to Catalog"))
	status, catalog := s.Get("http://" + ingress + "/catalogue")
	Expect(status).To(Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", ingress, status))
	Expect(strings.Contains(catalog, "/catalogue/")).To(Equal(true), fmt.Sprintf("Webpage found is NOT the Catalog %v", catalog))
	var catalogItems []CatalogItem
	json.Unmarshal([]byte(catalog), &catalogItems)
	Expect(len(catalogItems)).To(Not(Equal(0)), fmt.Sprint("Catalog page returned no items"))
	//return catalogItems[0]
	return Catalog{Item: catalogItems}
}

// AddToCart adds an item to the cart
func (s *SockShop) AddToCart(item CatalogItem) {
	ingress := s.Ingress
	//cartUrl := fmt.Sprintf("http://%v/cart", ingress)
	cartUrl := fmt.Sprintf("http://%v/carts/%v/items", ingress, s.username)
	cartBody := fmt.Sprintf(`{"itemId": "%v","unitPrice": "%v"}`, item.Id, item.Price)
	status, _ := s.Post(cartUrl, "application/json", strings.NewReader(cartBody))
	pkg.Log(Info, fmt.Sprintf("Finished adding to cart %v status: %v", cartBody, status))
	Expect(status).To(Equal(201), fmt.Sprintf("POST %v failed with status %v", cartUrl, status))
}

// CheckCart makes sure that the added item in the cart is present
func (s *SockShop) CheckCart(item CatalogItem, quantity int) {
	ingress := s.Ingress
	cartUrl := fmt.Sprintf("http://%v/carts/%v/items", ingress, s.username)
	status, cart := s.Get(cartUrl)
	Expect(status).To(Equal(200), fmt.Sprintf("GET %v failed with status %v", cartUrl, status))
	pkg.Log(Info, fmt.Sprintf("Retreived cart: %v", cart))
	var cartItems []CartItem
	json.Unmarshal([]byte(cart), &cartItems)
	foundItem := func() bool {
		for _, cartItem := range cartItems {
			if cartItem.ItemId == item.Id && cartItem.Quantity == quantity && cartItem.UnitPrice == item.Price {
				return true
			}
		}
		return false
	}
	Expect(foundItem()).To(BeTrue(), fmt.Sprintf("Could not find %v in cart", item.Name))
}

// GetCartItems gathers all cart items
func (s *SockShop) GetCartItems() []CartItem {
	ingress := s.Ingress
	pkg.Log(Info, fmt.Sprint("Gathering cart items"))
	cartUrl := fmt.Sprintf("http://%v/carts/%v/items", ingress, s.username)
	status, toDelCart := s.Get(cartUrl)

	Expect(status).To(Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", cartUrl, status))
	var toDelCartItems []CartItem
	json.Unmarshal([]byte(toDelCart), &toDelCartItems)
	return toDelCartItems
}

// DeleteCartItems deletes all cart items
func (s *SockShop) DeleteCartItems(items []CartItem) {
	ingress := s.Ingress
	cartUrl := fmt.Sprintf("http://%v/carts/%v/items", ingress, s.username)
	pkg.Log(Info, fmt.Sprintf("Deleting cart items: %v", items))
	for _, item := range items {
		status, cartDel := s.Delete(cartUrl + "/" + item.ItemId)
		Expect(status).To(Or(Equal(202)), fmt.Sprintf("Cart item %v not successfully deleted, response: %v status: %v", item.ItemId, cartDel, status))
	}
}

// CheckCartEmpty checks whether the cart has no items
func (s *SockShop) CheckCartEmpty() {
	ingress := s.Ingress
	cartUrl := fmt.Sprintf("http://%v/carts/%v/items", ingress, s.username)
	status, cart := s.Get(cartUrl)
	Expect(status).To(Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", cartUrl, status))
	var cartItems []CartItem
	json.Unmarshal([]byte(cart), &cartItems)
	Expect(len(cartItems)).To(Equal(0), fmt.Sprint("Cart page contained lingering items"))
}

// AccessPath ensures the given path is accessible
func (s *SockShop) AccessPath(path, expectedString string) {
	// move to cart page
	ingress := s.Ingress
	pkg.Log(Info, fmt.Sprint("Moving into the cart page"))
	basketUrl := fmt.Sprintf("http://%v/%v", ingress, path)
	status, basket := s.Get(basketUrl)
	Expect(status).To(Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", basketUrl, status))
	Expect(basket).To(ContainSubstring(expectedString), fmt.Sprintf("website found is NOT the shopping cart"))
}

// ChangeAddress changes the address for the provided user
func (s *SockShop) ChangeAddress(username string) {
	ingress := s.Ingress
	pkg.Log(Info, fmt.Sprint("Attempting to change address to 100 Oracle Pkwy, Redwood City, CA 94065"))
	addressData := fmt.Sprintf(`{"userID": "%v", "number":"100", "street":"Oracle Pkwy", "city":"Redwood City", "postcode":"94065", "country":"USA"}`, username)
	addressUrl := fmt.Sprintf("http://%v/addresses", ingress)
	status, address := s.Post(addressUrl, "application/json", strings.NewReader(addressData))
	Expect(status).To(Equal(200), fmt.Sprintf("POST %v returns status %v expected 200", addressUrl, status))
	var addressId Id
	json.Unmarshal([]byte(address), addressId)

	if addSplit := strings.Split(addressId.id, ":"); len(addSplit) == 2 {
		Expect(addSplit[0]).To(Equal(username), fmt.Sprintf("Incorrect ID expected %v and received %v", username, addSplit[0]))
		integ, err := strconv.Atoi(addSplit[1])
		Expect((integ > 0 && err == nil)).To(BeTrue(), fmt.Sprintf("Incorrect ID expected a positive integer and received %v", addSplit[1]))
	}
	pkg.Log(Info, fmt.Sprintf("Address: %v has been implemented with id", address))
}

// ChangePayment changes the form of payment
func (s *SockShop) ChangePayment() {
	// change payment
	ingress := s.Ingress
	pkg.Log(Info, fmt.Sprint("Attempting to change payment to 0000111122223333"))

	cardData := fmt.Sprintf(`{"userID": "%v", "longNum":"00001111222223333", "expires":"01/23", "ccv":"123"}`, s.username)

	cardUrl := fmt.Sprintf("http://%v/cards", ingress)
	status, card := s.Post(cardUrl, "application/json", strings.NewReader(cardData))
	Expect(status).To(Equal(200), fmt.Sprintf("POST %v returns status %v expected 200", cardUrl, status))
	pkg.Log(Info, fmt.Sprintf("Card with ID: %v has been implemented", card))
}

// GetOrders retrieves the orders
func (s *SockShop) GetOrders() {
	ingress := s.Ingress
	pkg.Log(Info, fmt.Sprint("Attempting to locate orders"))
	ordersUrl := fmt.Sprintf("http://%v/orders", ingress)
	status, orders := s.Get(ordersUrl)
	Expect(status).To(Equal(201), fmt.Sprintf("Get %v returns status %v expected 201", ordersUrl, status))
	pkg.Log(Info, fmt.Sprintf("Orders: %v have been retrieved", orders))
}
