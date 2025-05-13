package cloudinfo

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCloudInfo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudInfo Suite")
}

var _ = Describe("CloudInfo", func() {
	Describe("detectProvider", func() {
		It("should detect GCP provider", func() {
			labels := map[string]string{"cloud.google.com/gke-nodepool": "default-pool"}
			Expect(detectProvider(labels)).To(Equal("gcp"))
		})

		It("should detect AWS provider", func() {
			labels := map[string]string{"eks.amazonaws.com/nodegroup": "ng1"}
			Expect(detectProvider(labels)).To(Equal("aws"))
		})

		It("should detect Azure provider", func() {
			labels := map[string]string{"kubernetes.azure.com/role": "agent"}
			Expect(detectProvider(labels)).To(Equal("azure"))
		})

		It("should return unknown for unrecognized provider", func() {
			labels := map[string]string{"foo": "bar"}
			Expect(detectProvider(labels)).To(Equal("unknown"))
		})
	})

})

// Mock client.Client for DetectCloudEnvironment
type fakeClient struct {
	client.Client
	listFunc func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

func (f *fakeClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return f.listFunc(ctx, list, opts...)
}

var _ = Describe("DetectCloudEnvironment", func() {
	var (
		ctx context.Context
		fake *fakeClient
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("with GCP node", func() {
		BeforeEach(func() {
			fake = &fakeClient{
				listFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					nl, ok := list.(*corev1.NodeList)
					Expect(ok).To(BeTrue())
					nl.Items = []corev1.Node{{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"cloud.google.com/gke-nodepool": "pool",
								"topology.kubernetes.io/region": "us-central1",
								"topology.kubernetes.io/zone":   "us-central1-a",
							},
						},
					}}
					return nil
				},
			}
		})

		It("should detect GCP environment", func() {
			env, err := DetectCloudEnvironment(ctx, fake)
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(&CloudEnvironment{
				Provider: "gcp",
				Region:   "us-central1",
				Zone:     "us-central1-a",
			}))
		})
	})

	Context("with AWS node", func() {
		BeforeEach(func() {
			fake = &fakeClient{
				listFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					nl, ok := list.(*corev1.NodeList)
					Expect(ok).To(BeTrue())
					nl.Items = []corev1.Node{{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"eks.amazonaws.com/nodegroup": "ng1",
								"topology.kubernetes.io/region": "us-east-1",
								"topology.kubernetes.io/zone":   "us-east-1a",
							},
						},
					}}
					return nil
				},
			}
		})

		It("should detect AWS environment", func() {
			env, err := DetectCloudEnvironment(ctx, fake)
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(&CloudEnvironment{
				Provider: "aws",
				Region:   "us-east-1",
				Zone:     "us-east-1a",
			}))
		})
	})

	Context("with Azure node", func() {
		BeforeEach(func() {
			fake = &fakeClient{
				listFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					nl, ok := list.(*corev1.NodeList)
					Expect(ok).To(BeTrue())
					nl.Items = []corev1.Node{{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"kubernetes.azure.com/role": "agent",
								"topology.kubernetes.io/region": "westeurope",
								"topology.kubernetes.io/zone":   "1",
							},
						},
					}}
					return nil
				},
			}
		})

		It("should detect Azure environment", func() {
			env, err := DetectCloudEnvironment(ctx, fake)
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(&CloudEnvironment{
				Provider: "azure",
				Region:   "westeurope",
				Zone:     "1",
			}))
		})
	})

	Context("with unknown provider", func() {
		BeforeEach(func() {
			fake = &fakeClient{
				listFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					nl, ok := list.(*corev1.NodeList)
					Expect(ok).To(BeTrue())
					nl.Items = []corev1.Node{{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
								"topology.kubernetes.io/region": "some-region",
								"topology.kubernetes.io/zone":   "some-zone",
							},
						},
					}}
					return nil
				},
			}
		})

		It("should detect unknown environment", func() {
			env, err := DetectCloudEnvironment(ctx, fake)
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(Equal(&CloudEnvironment{
				Provider: "unknown",
				Region:   "some-region",
				Zone:     "some-zone",
			}))
		})
	})

	Context("with no nodes", func() {
		BeforeEach(func() {
			fake = &fakeClient{
				listFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					nl, ok := list.(*corev1.NodeList)
					Expect(ok).To(BeTrue())
					nl.Items = []corev1.Node{}
					return nil
				},
			}
		})

		It("should return an error", func() {
			env, err := DetectCloudEnvironment(ctx, fake)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no nodes found"))
			Expect(env).To(BeNil())
		})
	})

	Context("with list error", func() {
		BeforeEach(func() {
			fake = &fakeClient{
				listFunc: func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					return errors.New("fail-list")
				},
			}
		})

		It("should return an error", func() {
			env, err := DetectCloudEnvironment(ctx, fake)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to list nodes"))
			Expect(env).To(BeNil())
		})
	})
})

// End of test file
