import pytest
import sys
import os

# Add the parent directory to the path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from features import compute_features, FeatureEngineering


class TestFeatureEngineering:
    """Test feature engineering for lead scoring"""

    def test_compute_features_basic(self):
        """Test basic feature computation"""
        lead_data = {
            'name': 'Nguyen Van A',
            'email': 'nguyenvana@example.com',
            'phone': '0909123456',
            'company': 'Example Corp',
            'utm_source': 'google',
            'utm_campaign': 'spring_sale',
            'page_views': 5,
            'time_on_site': 300,
            'form_submissions': 1,
        }

        features = compute_features(lead_data)

        assert 'has_email' in features
        assert features['has_email'] == 1
        assert 'has_phone' in features
        assert features['has_phone'] == 1
        assert 'email_domain_score' in features
        assert 'phone_valid' in features

    def test_email_validation(self):
        """Test email validation features"""
        test_cases = [
            ('test@gmail.com', True),
            ('test@yahoo.com', True),
            ('test@company.com', True),
            ('test@unknown.xyz', False),
            ('invalid', False),
            ('', False),
        ]

        fe = FeatureEngineering()

        for email, expected_good in test_cases:
            score = fe._score_email_domain(email)
            if expected_good:
                assert score >= 0.5, f"Expected good domain score for {email}"
            else:
                assert score < 0.5, f"Expected low domain score for {email}"

    def test_phone_validation(self):
        """Test phone validation features"""
        test_cases = [
            ('0909123456', True),  # Vietnamese mobile
            ('0912345678', True),
            ('84909123456', True),  # With country code
            ('1234567890', False),  # Too short
            ('abcdefghij', False),  # Not digits
            ('', False),
        ]

        fe = FeatureEngineering()

        for phone, expected_valid in test_cases:
            is_valid = fe._validate_phone(phone)
            assert is_valid == expected_valid, f"Phone {phone}: expected {expected_valid}, got {is_valid}"

    def test_utm_scoring(self):
        """Test UTM parameter scoring"""
        fe = FeatureEngineering()

        # Known good UTM sources
        google_score = fe._score_utm_source('google')
        facebook_score = fe._score_utm_source('facebook')
        linkedin_score = fe._score_utm_source('linkedin')

        assert google_score > 0.5
        assert facebook_score > 0.5
        assert linkedin_score > 0.5

        # Unknown source should have lower score
        unknown_score = fe._score_utm_source('random_source')
        assert unknown_score < google_score

    def test_engagement_features(self):
        """Test engagement-based features"""
        fe = FeatureEngineering()

        # High engagement
        high_engagement = {
            'page_views': 20,
            'time_on_site': 600,
            'form_submissions': 3,
        }
        features = fe.compute_engagement_features(high_engagement)
        assert features['engagement_score'] >= 0.7

        # Low engagement
        low_engagement = {
            'page_views': 1,
            'time_on_site': 10,
            'form_submissions': 0,
        }
        features = fe.compute_engagement_features(low_engagement)
        assert features['engagement_score'] < 0.3

    def test_company_features(self):
        """Test company-based features"""
        fe = FeatureEngineering()

        # Known company
        known_company = fe._score_company('vietnam technology solution')
        assert known_company > 0.5

        # Unknown company
        unknown_company = fe._score_company('xyz abc def')
        assert unknown_company < 0.5

    def test_missing_fields(self):
        """Test handling of missing fields"""
        lead_data = {}

        fe = FeatureEngineering()
        features = fe.compute_all_features(lead_data)

        # Should have default values for missing fields
        assert features['has_email'] == 0
        assert features['has_phone'] == 0
        assert features['has_company'] == 0

    def test_feature_vector_shape(self):
        """Test that feature vector has correct shape"""
        lead_data = {
            'name': 'Test User',
            'email': 'test@example.com',
            'phone': '0909123456',
            'company': 'Test Corp',
            'utm_source': 'google',
            'page_views': 10,
            'time_on_site': 300,
            'form_submissions': 2,
        }

        fe = FeatureEngineering()
        features = fe.compute_all_features(lead_data)

        # Feature vector should be a list of floats
        assert isinstance(features, (list, tuple))
        for f in features:
            assert isinstance(f, (int, float))


class TestFeatureEdgeCases:
    """Test edge cases in feature engineering"""

    def test_unicode_handling(self):
        """Test handling of unicode characters"""
        lead_data = {
            'name': 'Nguyễn Văn A',
            'email': 'test@example.com',
            'phone': '0909123456',
        }

        fe = FeatureEngineering()
        # Should not raise
        features = fe.compute_all_features(lead_data)
        assert features is not None

    def test_very_long_values(self):
        """Test handling of very long input values"""
        lead_data = {
            'name': 'A' * 1000,
            'email': 'test@example.com',
            'phone': '0909123456',
        }

        fe = FeatureEngineering()
        features = fe.compute_all_features(lead_data)
        assert features is not None

    def test_special_characters(self):
        """Test handling of special characters"""
        lead_data = {
            'name': 'John <script>alert(1)</script> Doe',
            'email': 'test@example.com',
            'phone': '0909123456',
        }

        fe = FeatureEngineering()
        features = fe.compute_all_features(lead_data)
        assert features is not None

    def test_numeric_phone(self):
        """Test handling of numeric strings"""
        # Some systems might send phone as number
        phone = 909123456

        fe = FeatureEngineering()
        is_valid = fe._validate_phone(str(phone))
        assert is_valid == True


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
