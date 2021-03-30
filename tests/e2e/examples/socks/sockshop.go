// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/onsi/gomega"
)

// Constants for log levels
const (
	Debug = pkg.Debug
	Info  = pkg.Info
	Error = pkg.Error
)

// SockShop encapsulates all testing information related to interactions with the sock shop app
type SockShop struct {
	Cookies    []*http.Cookie
	Ingress    string
	username   string
	password   string
	hostHeader string
}

// CatalogItem represents the information for an item in the catalog
type CatalogItem struct {
	Count       int      `json:"count"`
	Description string   `json:"description"`
	ID          string   `json:"id"`
	ImageURL    string   `json:"imageurl"`
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
	ItemID    string  `json:"itemId"`
	Quantity  int     `json:"quantity"`
	UnitPrice float32 `json:"unitPrice"`
}

// Cart contains the items of a cart
type Cart struct {
	Item []CartItem
}

// ID is an id struct
type ID struct {
	id string
}

// NewSockShop creates a new sockshop instance
func NewSockShop(username, password, ingress string) SockShop {
	var sockShop SockShop
	sockShop.username = username
	sockShop.password = password
	sockShop.Ingress = ingress
	return sockShop
}

// SetHostHeader sets the ingress host
func (s *SockShop) SetHostHeader(host string) {
	s.hostHeader = host
}

// GetHostHeader returns the ingress host
func (s *SockShop) GetHostHeader() string {
	return s.hostHeader
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
func (s *SockShop) RegisterUser(body string, hostname string) {
	url := fmt.Sprintf("https://%v/register", hostname)
	status, register := pkg.PostWithHostHeader(url, "application/json", s.hostHeader, strings.NewReader(body))
	pkg.Log(Info, fmt.Sprintf("Finished register %v status: %v", register, status))
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", hostname, status))
	gomega.Expect(strings.Contains(register, "username")).To(gomega.Equal(true), fmt.Sprintf("Cannot register %v", register))
}

// ConnectToCatalog connects to the catalog page
func (s *SockShop) ConnectToCatalog(hostname string) string {
	// connect to catalog
	pkg.Log(Info, fmt.Sprint("Connecting to Catalog"))
	status, webpage := s.Get("https://" + hostname + "/catalogue")
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", hostname, status))
	gomega.Expect(strings.Contains(webpage, "/catalogue/")).To(gomega.Equal(true), fmt.Sprintf("Webpage found is NOT the Catalog %v", webpage))
	return webpage
}

// VerifyCatalogItems gets catalog items and accesses detail page
func (s *SockShop) VerifyCatalogItems(webpage string) {
	//ingress := s.Ingress
	var items []CatalogItem
	json.Unmarshal([]byte(webpage), &items)
	gomega.Expect(len(items)).To(gomega.Not(gomega.Equal(0)), fmt.Sprint("Catalog page returned no items"))
}

// GetCatalogItem retrieves the first catalog item
func (s *SockShop) GetCatalogItem(hostname string) Catalog {
	pkg.Log(Info, fmt.Sprint("Connecting to Catalog"))
	status, catalog := s.Get("https://" + hostname + "/catalogue")
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", hostname, status))
	gomega.Expect(strings.Contains(catalog, "/catalogue/")).To(gomega.Equal(true), fmt.Sprintf("Webpage found is NOT the Catalog %v", catalog))
	var catalogItems []CatalogItem
	json.Unmarshal([]byte(catalog), &catalogItems)
	gomega.Expect(len(catalogItems)).To(gomega.Not(gomega.Equal(0)), fmt.Sprint("Catalog page returned no items"))
	//return catalogItems[0]
	return Catalog{Item: catalogItems}
}

// AddToCart adds an item to the cart
func (s *SockShop) AddToCart(item CatalogItem, hostname string) {
	//cartURL := fmt.Sprintf("https://%v/cart", ingress)
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	cartBody := fmt.Sprintf(`{"itemId": "%v","unitPrice": "%v"}`, item.ID, item.Price)
	status, _ := s.Post(cartURL, "application/json", strings.NewReader(cartBody))
	pkg.Log(Info, fmt.Sprintf("Finished adding to cart %v status: %v", cartBody, status))
	gomega.Expect(status).To(gomega.Equal(201), fmt.Sprintf("POST %v failed with status %v", cartURL, status))
}

// CheckCart makes sure that the added item in the cart is present
func (s *SockShop) CheckCart(item CatalogItem, quantity int, hostname string) {
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	status, cart := s.Get(cartURL)
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("GET %v failed with status %v", cartURL, status))
	pkg.Log(Info, fmt.Sprintf("Retreived cart: %v", cart))
	var cartItems []CartItem
	json.Unmarshal([]byte(cart), &cartItems)
	foundItem := func() bool {
		for _, cartItem := range cartItems {
			if cartItem.ItemID == item.ID && cartItem.Quantity == quantity && cartItem.UnitPrice == item.Price {
				return true
			}
		}
		return false
	}
	gomega.Expect(foundItem()).To(gomega.BeTrue(), fmt.Sprintf("Could not find %v in cart", item.Name))
}

// GetCartItems gathers all cart items
func (s *SockShop) GetCartItems(hostname string) []CartItem {
	pkg.Log(Info, fmt.Sprint("Gathering cart items"))
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	status, toDelCart := s.Get(cartURL)

	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", cartURL, status))
	var toDelCartItems []CartItem
	json.Unmarshal([]byte(toDelCart), &toDelCartItems)
	return toDelCartItems
}

// DeleteCartItems deletes all cart items
func (s *SockShop) DeleteCartItems(items []CartItem, hostname string) {
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	pkg.Log(Info, fmt.Sprintf("Deleting cart items: %v", items))
	for _, item := range items {
		status, cartDel := s.Delete(cartURL + "/" + item.ItemID)
		gomega.Expect(status).To(gomega.Or(gomega.Equal(202)), fmt.Sprintf("Cart item %v not successfully deleted, response: %v status: %v", item.ItemID, cartDel, status))
	}
}

// CheckCartEmpty checks whether the cart has no items
func (s *SockShop) CheckCartEmpty(hostname string) {
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	status, cart := s.Get(cartURL)
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", cartURL, status))
	var cartItems []CartItem
	json.Unmarshal([]byte(cart), &cartItems)
	gomega.Expect(len(cartItems)).To(gomega.Equal(0), fmt.Sprint("Cart page contained lingering items"))
}

// AccessPath ensures the given path is accessible
func (s *SockShop) AccessPath(path, expectedString string, hostname string) {
	// move to cart page
	pkg.Log(Info, fmt.Sprint("Moving into the cart page"))
	basketURL := fmt.Sprintf("https://%v/%v", hostname, path)
	status, basket := s.Get(basketURL)
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", basketURL, status))
	gomega.Expect(basket).To(gomega.ContainSubstring(expectedString), fmt.Sprintf("website found is NOT the shopping cart"))
}

// ChangeAddress changes the address for the provided user
func (s *SockShop) ChangeAddress(username string, hostname string) {
	pkg.Log(Info, fmt.Sprint("Attempting to change address to 100 Oracle Pkwy, Redwood City, CA 94065"))
	addressData := fmt.Sprintf(`{"userID": "%v", "number":"100", "street":"Oracle Pkwy", "city":"Redwood City", "postcode":"94065", "country":"USA"}`, username)
	addressURL := fmt.Sprintf("https://%v/addresses", hostname)
	status, address := s.Post(addressURL, "application/json", strings.NewReader(addressData))
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("POST %v returns status %v expected 200", addressURL, status))
	var addressID *ID
	json.Unmarshal([]byte(address), &addressID)

	if addSplit := strings.Split(addressID.id, ":"); len(addSplit) == 2 {
		gomega.Expect(addSplit[0]).To(gomega.Equal(username), fmt.Sprintf("Incorrect ID expected %v and received %v", username, addSplit[0]))
		integ, err := strconv.Atoi(addSplit[1])
		gomega.Expect((integ > 0 && err == nil)).To(gomega.BeTrue(), fmt.Sprintf("Incorrect ID expected a positive integer and received %v", addSplit[1]))
	}
	pkg.Log(Info, fmt.Sprintf("Address: %v has been implemented with id", address))
}

// ChangePayment changes the form of payment
func (s *SockShop) ChangePayment(hostname string) {
	// change payment
	pkg.Log(Info, fmt.Sprint("Attempting to change payment to 0000111122223333"))

	cardData := fmt.Sprintf(`{"userID": "%v", "longNum":"00001111222223333", "expires":"01/23", "ccv":"123"}`, s.username)

	cardURL := fmt.Sprintf("https://%v/cards", hostname)
	status, card := s.Post(cardURL, "application/json", strings.NewReader(cardData))
	gomega.Expect(status).To(gomega.Equal(200), fmt.Sprintf("POST %v returns status %v expected 200", cardURL, status))
	pkg.Log(Info, fmt.Sprintf("Card with ID: %v has been implemented", card))
}

// GetOrders retrieves the orders
func (s *SockShop) GetOrders(hostname string) {
	pkg.Log(Info, fmt.Sprint("Attempting to locate orders"))
	ordersURL := fmt.Sprintf("https://%v/orders", hostname)
	status, orders := s.Get(ordersURL)
	gomega.Expect(status).To(gomega.Equal(201), fmt.Sprintf("Get %v returns status %v expected 201", ordersURL, status))
	pkg.Log(Info, fmt.Sprintf("Orders: %v have been retrieved", orders))
}

// undeploySockShopApplication undeploys the sock shop application
func undeploySockShopApplication() error {
	err := pkg.DeleteNamespace("sockshop")
	if err != nil {
		return err
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace("sockshop")
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
	return nil
}
