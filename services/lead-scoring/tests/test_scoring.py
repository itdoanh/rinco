import pytest
import sys
import os
import numpy as np
from unittest.mock import Mock, patch

# Add the parent directory to the path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from scoring import LeadScorer, ScorerConfig


class TestLeadScorer:
    """Test lead scoring inference"""

    @pytest.fixture
    def scorer(self):
        """Create a scorer instance"""
        config = ScorerConfig(
            model_path=None,  # We'll use mock model
            threshold=0.5,
            min_features=5,
        )
        return LeadScorer(config)

    @pytest.fixture
    def sample_features(self):
        """Sample feature vector"""
        return np.array([1.0, 1.0, 0.8, 0.7, 0.9, 0.6, 0.5, 0.8, 0.4, 0.3])

    def test_scorer_initialization(self, scorer):
        """Test scorer initializes correctly"""
        assert scorer is not None
        assert scorer.config.threshold == 0.5

    def test_predict_returns_score(self, scorer, sample_features):
        """Test prediction returns a score"""
        score = scorer.predict(sample_features)

        assert isinstance(score, (int, float))
        assert 0 <= score <= 1

    def test_predict_proba(self, scorer, sample_features):
        """Test probability prediction"""
        proba = scorer.predict_proba(sample_features)

        assert isinstance(proba, (list, tuple))
        assert len(proba) == 2  # [prob_not_hot, prob_hot]
        assert abs(sum(proba) - 1.0) < 0.001  # Should sum to 1

    def test_predict_with_dict_features(self, scorer):
        """Test prediction with dictionary features"""
        features = {
            'has_email': 1.0,
            'has_phone': 1.0,
            'email_domain_score': 0.8,
            'phone_valid': 1.0,
            'utm_source_score': 0.7,
            'company_score': 0.6,
            'engagement_score': 0.5,
            'time_score': 0.8,
            'behavior_score': 0.4,
            'intent_score': 0.3,
        }

        score = scorer.predict_from_dict(features)
        assert isinstance(score, (int, float))
        assert 0 <= score <= 1

    def test_threshold_classification(self, scorer):
        """Test threshold-based classification"""
        # High score should be hot
        high_features = np.array([1.0] * 10)
        is_hot = scorer.is_hot(high_features)
        assert is_hot == True

        # Low score should be cold
        low_features = np.array([0.1] * 10)
        is_hot = scorer.is_hot(low_features)
        assert is_hot == False

    def test_batch_prediction(self, scorer):
        """Test batch prediction"""
        features_batch = np.array([
            [1.0, 1.0, 0.8, 0.7, 0.9, 0.6, 0.5, 0.8, 0.4, 0.3],
            [0.0, 0.0, 0.2, 0.1, 0.1, 0.2, 0.1, 0.2, 0.1, 0.1],
            [0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5],
        ])

        scores = scorer.predict_batch(features_batch)

        assert len(scores) == 3
        for score in scores:
            assert 0 <= score <= 1

    def test_invalid_features_raises(self, scorer):
        """Test that invalid features raise appropriate errors"""
        # Wrong number of features
        wrong_features = np.array([1.0, 2.0])

        with pytest.raises(ValueError):
            scorer.predict(wrong_features)

    def test_missing_features_raises(self, scorer):
        """Test that missing features raise errors"""
        # Fewer features than expected
        incomplete_features = np.array([1.0] * 5)

        with pytest.raises(ValueError):
            scorer.predict(incomplete_features)


class TestScorerConfig:
    """Test scorer configuration"""

    def test_default_config(self):
        """Test default configuration"""
        config = ScorerConfig()

        assert config.threshold == 0.5
        assert config.min_features == 5
        assert config.model_path is None

    def test_custom_config(self):
        """Test custom configuration"""
        config = ScorerConfig(
            threshold=0.7,
            min_features=10,
            model_path='/path/to/model.pkl',
        )

        assert config.threshold == 0.7
        assert config.min_features == 10
        assert config.model_path == '/path/to/model.pkl'

    def test_config_validation(self):
        """Test configuration validation"""
        # Valid threshold
        config = ScorerConfig(threshold=0.8)
        assert config.threshold == 0.8

        # Threshold should be between 0 and 1
        config = ScorerConfig(threshold=1.5)
        # Implementation should clamp or raise


class TestScoringEdgeCases:
    """Test edge cases in scoring"""

    def test_all_zeros_features(self):
        """Test scoring with all zero features"""
        scorer = LeadScorer()
        features = np.zeros(10)

        score = scorer.predict(features)
        # Should not crash, even with zero features
        assert isinstance(score, (int, float))

    def test_all_ones_features(self):
        """Test scoring with all ones features"""
        scorer = LeadScorer()
        features = np.ones(10)

        score = scorer.predict(features)
        assert 0 <= score <= 1

    def test_extreme_values(self):
        """Test scoring with extreme values"""
        scorer = LeadScorer()
        features = np.array([1e10, -1e10, 1e10, -1e10, 1e10, -1e10, 1e10, -1e10, 1e10, -1e10])

        # Should handle extreme values gracefully
        try:
            score = scorer.predict(features)
            assert isinstance(score, (int, float))
        except (OverflowError, ValueError):
            # Acceptable if it raises on extreme values
            pass

    def test_nan_handling(self):
        """Test handling of NaN values"""
        scorer = LeadScorer()
        features = np.array([1.0, np.nan, 0.5, np.nan, 0.8, 0.6, 0.5, 0.8, 0.4, 0.3])

        # Should handle NaN values
        try:
            score = scorer.predict(features)
            assert isinstance(score, (int, float))
        except (ValueError, TypeError):
            # Acceptable if it raises on NaN
            pass


class TestIntegration:
    """Integration tests for the scoring pipeline"""

    def test_full_pipeline(self):
        """Test the full scoring pipeline"""
        # This would typically test:
        # 1. Feature engineering
        # 2. Model loading
        # 3. Prediction
        # 4. Result formatting

        # For now, just test that components exist
        from features import compute_features
        from scoring import LeadScorer

        # Sample lead data
        lead_data = {
            'name': 'Test Lead',
            'email': 'test@example.com',
            'phone': '0909123456',
            'company': 'Test Corp',
            'utm_source': 'google',
            'page_views': 10,
            'time_on_site': 300,
            'form_submissions': 2,
        }

        # Compute features
        features = compute_features(lead_data)

        # Score
        scorer = LeadScorer()
        score = scorer.predict_from_dict(features)

        # Verify result
        assert 0 <= score <= 1


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
