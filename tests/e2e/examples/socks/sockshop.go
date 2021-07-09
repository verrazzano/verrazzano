// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/gomega"
)

// Constants for log levels
const (
	Debug = pkg.Debug
	Info  = pkg.Info
	Error = pkg.Error
)

// SockShop encapsulates all testing information related to interactions with the sock shop app
type SockShop struct {
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
func (s *SockShop) Post(url, contentType string, body io.Reader) (*pkg.HTTPResponse, error) {
	return pkg.PostWithHostHeader(url, contentType, s.hostHeader, body)
}

// Get is a wrapper function for HTTP request with cookies GET
func (s *SockShop) Get(url string) (*pkg.HTTPResponse, error) {
	return pkg.GetWebPage(url, s.hostHeader)
}

// Delete is a wrapper function for HTTP request with cookies DELETE
func (s *SockShop) Delete(url string) (*pkg.HTTPResponse, error) {
	return pkg.Delete(url, s.hostHeader)
}

// RegisterUser interacts with sock shop to create a user
func (s *SockShop) RegisterUser(body string, hostname string) bool {
	url := fmt.Sprintf("https://%v/register", hostname)
	resp, err := pkg.PostWithHostHeader(url, "application/json", s.hostHeader, strings.NewReader(body))
<<<<<<< HEAD
	Expect(err).ShouldNot(HaveOccurred())
=======
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
>>>>>>> 6ae4e52e... Updated projectCmd
	pkg.Log(Info, fmt.Sprintf("Finished register %s status: %v", resp.Body, resp.StatusCode))
	return (resp.StatusCode == http.StatusOK) && (strings.Contains(string(resp.Body), "username"))
}

<<<<<<< HEAD
// GetCatalogItems retrieves the catalog items
func (s *SockShop) GetCatalogItems(hostname string) (*pkg.HTTPResponse, error) {
	pkg.Log(Info, "Connecting to Catalog")
	url := "https://" + hostname + "/catalogue"
	return s.Get(url)
=======
// ConnectToCatalog connects to the catalog page
func (s *SockShop) ConnectToCatalog(hostname string) string {
	// connect to catalog
	pkg.Log(Info, "Connecting to Catalog")
	url := "https://" + hostname + "/catalogue"
	resp, err := s.Get(url)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK), fmt.Sprintf("GET %v returns status %v expected 200", hostname, resp.StatusCode))
	gomega.Expect(strings.Contains(string(resp.Body), "/catalogue/")).To(gomega.Equal(true), fmt.Sprintf("Webpage found is NOT the Catalog %s", resp.Body))
	return string(resp.Body)
}

// VerifyCatalogItems gets catalog items and accesses detail page
func (s *SockShop) VerifyCatalogItems(webpage string) {
	//ingress := s.Ingress
	var items []CatalogItem
	json.Unmarshal([]byte(webpage), &items)
	gomega.Expect(len(items)).To(gomega.Not(gomega.Equal(0)), "Catalog page returned no items")
}

// GetCatalogItem retrieves the first catalog item
func (s *SockShop) GetCatalogItem(hostname string) Catalog {
	pkg.Log(Info, "Connecting to Catalog")
	url := "https://" + hostname + "/catalogue"
	resp, err := s.Get(url)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", hostname, resp.StatusCode))
	gomega.Expect(strings.Contains(string(resp.Body), "/catalogue/")).To(gomega.Equal(true), fmt.Sprintf("Webpage found is NOT the Catalog %s", resp.Body))
	var catalogItems []CatalogItem
	json.Unmarshal(resp.Body, &catalogItems)
	gomega.Expect(len(catalogItems)).To(gomega.Not(gomega.Equal(0)), "Catalog page returned no items")
	return Catalog{Item: catalogItems}
>>>>>>> 6ae4e52e... Updated projectCmd
}

// AddToCart adds an item to the cart
func (s *SockShop) AddToCart(item CatalogItem, hostname string) (*pkg.HTTPResponse, error) {
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	cartBody := fmt.Sprintf(`{"itemId": "%v","unitPrice": "%v"}`, item.ID, item.Price)
<<<<<<< HEAD
	return s.Post(cartURL, "application/json", strings.NewReader(cartBody))
}

// CheckCart makes sure that the added item in the cart is present
func (s *SockShop) CheckCart(cartItems []CartItem, item CatalogItem, quantity int) {
	found := false
	for _, cartItem := range cartItems {
		if cartItem.ItemID == item.ID && cartItem.Quantity == quantity && cartItem.UnitPrice == item.Price {
			found = true
			break
=======
	resp, err := s.Post(cartURL, "application/json", strings.NewReader(cartBody))
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	pkg.Log(Info, fmt.Sprintf("Finished adding to cart %v status: %v", cartBody, resp.StatusCode))
	gomega.Expect(resp.StatusCode).To(gomega.Equal(201), fmt.Sprintf("POST %v failed with status %v", cartURL, resp.StatusCode))
}

// CheckCart makes sure that the added item in the cart is present
func (s *SockShop) CheckCart(item CatalogItem, quantity int, hostname string) {
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	resp, err := s.Get(cartURL)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(200), fmt.Sprintf("GET %v failed with status %v", cartURL, resp.StatusCode))
	pkg.Log(Info, fmt.Sprintf("Retreived cart: %s", resp.Body))
	var cartItems []CartItem
	json.Unmarshal(resp.Body, &cartItems)
	foundItem := func() bool {
		for _, cartItem := range cartItems {
			if cartItem.ItemID == item.ID && cartItem.Quantity == quantity && cartItem.UnitPrice == item.Price {
				return true
			}
>>>>>>> 6ae4e52e... Updated projectCmd
		}
	}
	Expect(found).To(BeTrue(), fmt.Sprintf("Could not find %v in cart", item.Name))
}

// GetCartItems gathers all cart items
<<<<<<< HEAD
func (s *SockShop) GetCartItems(hostname string) (*pkg.HTTPResponse, error) {
	pkg.Log(Info, "Gathering cart items")
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	return s.Get(cartURL)
=======
func (s *SockShop) GetCartItems(hostname string) []CartItem {
	pkg.Log(Info, "Gathering cart items")
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	resp, err := s.Get(cartURL)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", cartURL, resp.StatusCode))
	var toDelCartItems []CartItem
	json.Unmarshal(resp.Body, &toDelCartItems)
	return toDelCartItems
>>>>>>> 6ae4e52e... Updated projectCmd
}

// DeleteCartItems deletes all cart items
func (s *SockShop) DeleteCartItem(item CartItem, hostname string) (*pkg.HTTPResponse, error) {
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
<<<<<<< HEAD
	pkg.Log(Info, fmt.Sprintf("Deleting cart item: %v", item))
	return s.Delete(cartURL + "/" + item.ItemID)
}

// ChangeAddress changes the address for the provided user
func (s *SockShop) ChangeAddress(username string, hostname string) (*pkg.HTTPResponse, error) {
	pkg.Log(Info, "Attempting to change address to 100 Oracle Pkwy, Redwood City, CA 94065")
	addressData := fmt.Sprintf(`{"userID": "%v", "number":"100", "street":"Oracle Pkwy", "city":"Redwood City", "postcode":"94065", "country":"USA"}`, username)
	addressURL := fmt.Sprintf("https://%v/addresses", hostname)
	return s.Post(addressURL, "application/json", strings.NewReader(addressData))
}

// CheckAddress unmarshals the address from the HTTPResponse and validates that it is correctly formatted and contains the username
func (s *SockShop) CheckAddress(response *pkg.HTTPResponse, username string) {
	var addressID *ID
	json.Unmarshal(response.Body, &addressID)
=======
	pkg.Log(Info, fmt.Sprintf("Deleting cart items: %v", items))
	for _, item := range items {
		resp, err := s.Delete(cartURL + "/" + item.ItemID)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		gomega.Expect(resp.StatusCode).To(gomega.Equal(202), fmt.Sprintf("Cart item %v not successfully deleted, response: %s status: %v", item.ItemID, resp.Body, resp.StatusCode))
	}
}

// CheckCartEmpty checks whether the cart has no items
func (s *SockShop) CheckCartEmpty(hostname string) {
	cartURL := fmt.Sprintf("https://%v/carts/%v/items", hostname, s.username)
	resp, err := s.Get(cartURL)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(200), fmt.Sprintf("GET %s returns status %v expected 200", resp.Body, resp.StatusCode))
	var cartItems []CartItem
	json.Unmarshal(resp.Body, &cartItems)
	gomega.Expect(len(cartItems)).To(gomega.Equal(0), "Cart page contained lingering items")
}

// AccessPath ensures the given path is accessible
func (s *SockShop) AccessPath(path, expectedString string, hostname string) {
	// move to cart page
	pkg.Log(Info, "Moving into the cart page")
	basketURL := fmt.Sprintf("https://%v/%v", hostname, path)
	resp, err := s.Get(basketURL)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(200), fmt.Sprintf("GET %v returns status %v expected 200", basketURL, resp.StatusCode))
	gomega.Expect(string(resp.Body)).To(gomega.ContainSubstring(expectedString), "website found is NOT the shopping cart")
}

// ChangeAddress changes the address for the provided user
func (s *SockShop) ChangeAddress(username string, hostname string) {
	pkg.Log(Info, "Attempting to change address to 100 Oracle Pkwy, Redwood City, CA 94065")
	addressData := fmt.Sprintf(`{"userID": "%v", "number":"100", "street":"Oracle Pkwy", "city":"Redwood City", "postcode":"94065", "country":"USA"}`, username)
	addressURL := fmt.Sprintf("https://%v/addresses", hostname)
	resp, err := s.Post(addressURL, "application/json", strings.NewReader(addressData))
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(200), fmt.Sprintf("POST %v returns status %v expected 200", addressURL, resp.StatusCode))
	var addressID *ID
	json.Unmarshal(resp.Body, &addressID)
>>>>>>> 6ae4e52e... Updated projectCmd

	if addSplit := strings.Split(addressID.id, ":"); len(addSplit) == 2 {
		Expect(addSplit[0]).To(Equal(username), fmt.Sprintf("Incorrect ID expected %v and received %v", username, addSplit[0]))
		integ, err := strconv.Atoi(addSplit[1])
		Expect((integ > 0 && err == nil)).To(BeTrue(), fmt.Sprintf("Incorrect ID expected a positive integer and received %v", addSplit[1]))
	}
<<<<<<< HEAD
	pkg.Log(Info, fmt.Sprintf("Address: %s has been implemented with id", response.Body))
=======
	pkg.Log(Info, fmt.Sprintf("Address: %s has been implemented with id", resp.Body))
>>>>>>> 6ae4e52e... Updated projectCmd
}

// ChangePayment changes the form of payment
func (s *SockShop) ChangePayment(hostname string) (*pkg.HTTPResponse, error) {
	// change payment
	pkg.Log(Info, "Attempting to change payment to 0000111122223333")

	cardData := fmt.Sprintf(`{"userID": "%v", "longNum":"00001111222223333", "expires":"01/23", "ccv":"123"}`, s.username)

	cardURL := fmt.Sprintf("https://%v/cards", hostname)
<<<<<<< HEAD
	return s.Post(cardURL, "application/json", strings.NewReader(cardData))
=======
	resp, err := s.Post(cardURL, "application/json", strings.NewReader(cardData))
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(200), fmt.Sprintf("POST %v returns status %v expected 200", cardURL, resp.StatusCode))
	pkg.Log(Info, fmt.Sprintf("Card with ID: %s has been implemented", resp.Body))
>>>>>>> 6ae4e52e... Updated projectCmd
}

// GetOrders retrieves the orders
func (s *SockShop) GetOrders(hostname string) {
	pkg.Log(Info, "Attempting to locate orders")
	ordersURL := fmt.Sprintf("https://%v/orders", hostname)
	resp, err := s.Get(ordersURL)
<<<<<<< HEAD
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(201), fmt.Sprintf("Get %v returns status %v expected 201", ordersURL, resp.StatusCode))
	pkg.Log(Info, fmt.Sprintf("Orders: %s have been retrieved", resp.Body))
=======
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	gomega.Expect(resp.StatusCode).To(gomega.Equal(201), fmt.Sprintf("Get %v returns status %v expected 201", ordersURL, resp.StatusCode))
	pkg.Log(Info, fmt.Sprintf("Orders: %s have been retrieved", resp.Body))
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
>>>>>>> 6ae4e52e... Updated projectCmd
}
